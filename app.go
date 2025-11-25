package main

import (
	"context"
	"fmt"
	"log"

	"bcdl-app/backend/models"
	"bcdl-app/backend/playwright"
	"bcdl-app/backend/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	pwService  *playwright.Service
	scanner    *services.ScannerService
	downloader *services.DownloaderService
	scanCancel context.CancelFunc
}

// NewApp creates a new App application struct
func NewApp() *App {
	pwService := playwright.NewService()
	return &App{
		pwService:  pwService,
		scanner:    services.NewScannerService(pwService),
		downloader: services.NewDownloaderService(pwService),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize Playwright
	if err := a.pwService.Init(); err != nil {
		log.Printf("Failed to init Playwright: %v", err)
		runtime.EventsEmit(a.ctx, "log:error", fmt.Sprintf("Failed to init Playwright: %v", err))
	}
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	a.pwService.Close()
}

// ScanArtist scans a Bandcamp artist URL for albums
func (a *App) ScanArtist(url string) ([]models.Album, error) {
	log.Printf("ScanArtist called with URL: %q", url)

	// Run scan in a goroutine to avoid blocking the Wails runtime
	// This ensures events are dispatched to the frontend immediately
	go func() {
		// Create cancellable context
		scanCtx, cancel := context.WithCancel(context.Background())
		a.scanCancel = cancel
		defer func() {
			a.scanCancel = nil
		}()

		runtime.EventsEmit(a.ctx, "scan:start", url)

		albums, err := a.scanner.ScanArtist(scanCtx, url, func(album models.Album) {
			log.Printf("Emitting scan:album_found for %s", album.Title)
			runtime.EventsEmit(a.ctx, "scan:album_found", album)
		})

		if err != nil {
			if err == context.Canceled {
				log.Printf("Scan cancelled")
				runtime.EventsEmit(a.ctx, "scan:stopped", len(albums))
				return
			}
			log.Printf("Scan error: %v", err)
			runtime.EventsEmit(a.ctx, "scan:error", err.Error())
			return
		}

		log.Printf("Scan complete: found %d albums", len(albums))
		runtime.EventsEmit(a.ctx, "scan:complete", albums)
	}()

	return nil, nil
}

// StopScan cancels the currently running scan
func (a *App) StopScan() error {
	if a.scanCancel != nil {
		log.Printf("StopScan called, cancelling scan...")
		a.scanCancel()
		return nil
	}
	log.Printf("StopScan called but no scan is running")
	return fmt.Errorf("no scan is currently running")
}

// DownloadAlbum downloads a single album
func (a *App) DownloadAlbum(url string, downloadDir string, format string) error {
	runtime.EventsEmit(a.ctx, "download:start", url)

	progressCallback := func(msg string) {
		runtime.EventsEmit(a.ctx, "download:progress", map[string]string{
			"url":     url,
			"message": msg,
		})
	}

	err := a.downloader.DownloadAlbum(url, downloadDir, format, progressCallback)
	if err != nil {
		runtime.EventsEmit(a.ctx, "download:error", map[string]string{
			"url":   url,
			"error": err.Error(),
		})
		return err
	}

	runtime.EventsEmit(a.ctx, "download:complete", url)
	return nil
}

// SelectFolder opens a dialog to select a folder
func (a *App) SelectFolder() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Download Folder",
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}
