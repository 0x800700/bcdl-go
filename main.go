package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"bcdl-app/backend/models"
	"bcdl-app/backend/services"
)

// SSEBroker manages connected Server-Sent Events clients
type SSEBroker struct {
	clients    map[chan string]bool
	clientLock sync.Mutex
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[chan string]bool),
	}
}

func (b *SSEBroker) AddClient(ch chan string) {
	b.clientLock.Lock()
	defer b.clientLock.Unlock()
	b.clients[ch] = true
}

func (b *SSEBroker) RemoveClient(ch chan string) {
	b.clientLock.Lock()
	defer b.clientLock.Unlock()
	delete(b.clients, ch)
	close(ch)
}

func (b *SSEBroker) Broadcast(event string, data interface{}) {
	b.clientLock.Lock()
	defer b.clientLock.Unlock()

	var dataStr string
	switch v := data.(type) {
	case string:
		dataStr = v
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			log.Printf("SSE: JSON marshal error: %v", err)
			return
		}
		dataStr = string(bytes)
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, dataStr)

	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// Client channel full or slow, skip
		}
	}
}

type Server struct {
	scanner     *services.ScannerService
	downloader  *services.DownloaderService
	broker      *SSEBroker
	scanCancel  context.CancelFunc
	scanLock    sync.Mutex
	downloadDir string
}

func NewServer(downloadDir string) *Server {
	return &Server{
		scanner:     services.NewScannerService(),
		downloader:  services.NewDownloaderService(),
		broker:      NewSSEBroker(),
		downloadDir: downloadDir,
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	msgChan := make(chan string, 50)
	s.broker.AddClient(msgChan)
	defer s.broker.RemoveClient(msgChan)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			fmt.Fprint(w, msg)
			flusher.Flush()
		}
	}
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	s.scanLock.Lock()
	if s.scanCancel != nil {
		s.scanCancel()
	}
	scanCtx, cancel := context.WithCancel(context.Background())
	s.scanCancel = cancel
	s.scanLock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started", "url": req.URL})

	go func() {
		defer func() {
			s.scanLock.Lock()
			s.scanCancel = nil
			s.scanLock.Unlock()
		}()

		s.broker.Broadcast("scan:start", req.URL)

		albums, err := s.scanner.ScanArtist(scanCtx, req.URL, func(album models.Album) {
			s.broker.Broadcast("scan:album_found", album)
		})

		if err != nil {
			if err == context.Canceled {
				s.broker.Broadcast("scan:stopped", len(albums))
				return
			}
			s.broker.Broadcast("scan:error", err.Error())
			return
		}

		s.broker.Broadcast("scan:complete", albums)
	}()
}

func (s *Server) handleStopScan(w http.ResponseWriter, r *http.Request) {
	s.scanLock.Lock()
	defer s.scanLock.Unlock()

	if s.scanCancel != nil {
		s.scanCancel()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "no_active_scan"})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL    string `json:"url"`
		Format string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Invalid download request", http.StatusBadRequest)
		return
	}

	if req.Format == "" {
		req.Format = "flac"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "queued", "url": req.URL})

	go func() {
		log.Printf("Download started for %s (format: %s)", req.URL, req.Format)
		s.broker.Broadcast("download:start", req.URL)

		progressCallback := func(msg string) {
			log.Printf("Download progress [%s]: %s", req.URL, msg)
			s.broker.Broadcast("download:progress", map[string]string{
				"url":     req.URL,
				"message": msg,
			})
		}

		err := s.downloader.DownloadAlbum(req.URL, s.downloadDir, req.Format, progressCallback)
		if err != nil {
			log.Printf("Download FAILED for %s: %v", req.URL, err)
			s.broker.Broadcast("download:error", map[string]string{
				"url":   req.URL,
				"error": err.Error(),
			})
			return
		}

		log.Printf("Download SUCCESS for %s", req.URL)
		s.broker.Broadcast("download:complete", req.URL)
	}()
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"download_dir": s.downloadDir,
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	downloadDir := os.Getenv("DOWNLOAD_DIR")
	if downloadDir == "" {
		downloadDir = "/downloads"
		if _, err := os.Stat("/downloads"); os.IsNotExist(err) {
			downloadDir = "./downloads"
		}
	}
	os.MkdirAll(downloadDir, 0755)

	server := NewServer(downloadDir)
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/events", server.handleEvents)
	mux.HandleFunc("/api/scan", server.handleScan)
	mux.HandleFunc("/api/scan/stop", server.handleStopScan)
	mux.HandleFunc("/api/download", server.handleDownload)
	mux.HandleFunc("/api/config", server.handleConfig)

	// Static UI serving (fallback to index.html for SPA)
	distPath := "./frontend/dist"
	if _, err := os.Stat(distPath); err == nil {
		fs := http.FileServer(http.Dir(distPath))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			// If file exists, serve it
			path := filepath.Join(distPath, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				fs.ServeHTTP(w, r)
				return
			}
			// Fallback to index.html for SPA router
			http.ServeFile(w, r, filepath.Join(distPath, "index.html"))
		})
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<h2>Bandcamp Downloader Server</h2><p>Frontend dist not found at %s. API is running on /api/.</p>", distPath)
		})
	}

	log.Printf("Starting Bandcamp Downloader server on :%s, downloads: %s", port, downloadDir)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
