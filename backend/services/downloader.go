package services

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type DownloaderService struct {
	apiClient    *http.Client
	streamClient *http.Client
	gmSvc        *GuerrillaMailService
	tempEmailSvc *TempEmailService
}

func NewDownloaderService() *DownloaderService {
	return &DownloaderService{
		apiClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		streamClient: &http.Client{
			Timeout: 0, // No timeout for streaming large files
		},
		gmSvc:        NewGuerrillaMailService(),
		tempEmailSvc: NewTempEmailService(),
	}
}

// ProgressCallback receives live status messages
type ProgressCallback func(message string)

type parsedTralbum struct {
	Current struct {
		ID           int64   `json:"id"`
		Type         string  `json:"type"`
		Title        string  `json:"title"`
		Artist       string  `json:"artist"`
		DownloadPref int     `json:"download_pref"`
		RequireEmail int     `json:"require_email"`
		SetPrice     float64 `json:"set_price"`
		MinimumPrice float64 `json:"minimum_price"`
	} `json:"current"`
	FreeDownloadPage string `json:"freeDownloadPage"`
	URL              string `json:"url"`
	ItemType         string `json:"item_type"`
	ID               int64  `json:"id"`
}

type pagedataBlob struct {
	DigitalItems []struct {
		Title     string `json:"title"`
		Artist    string `json:"artist"`
		ItemType  string `json:"item_type"`
		ItemID    int64  `json:"item_id"`
		Downloads map[string]struct {
			URL         string `json:"url"`
			Description string `json:"description"`
			SizeMB      string `json:"size_mb"`
		} `json:"downloads"`
	} `json:"digital_items"`
}

var formatKeyMap = map[string][]string{
	"flac":    {"flac", "wav", "mp3-320", "mp3-v0"},
	"mp3-320": {"mp3-320", "mp3-v0", "flac"},
	"wav":     {"wav", "flac", "mp3-320"},
	"alac":    {"alac", "flac", "wav"},
	"aac-hi":  {"aac-hi", "mp3-320"},
	"vorbis":  {"vorbis", "mp3-320"},
	"mp3-v0":  {"mp3-v0", "mp3-320"},
}

// DownloadAlbum handles downloading via direct Bandcamp HTTP API
func (s *DownloaderService) DownloadAlbum(rawURL string, downloadDir string, format string, progress ProgressCallback) error {
	normalizedURL := strings.ReplaceAll(rawURL, "bandcamp.com//", "bandcamp.com/")
	log.Printf("Downloader: Fetching info for %s", normalizedURL)
	progress(fmt.Sprintf("Fetching release info: %s", normalizedURL))

	// 1. Fetch page HTML
	req, err := http.NewRequest("GET", normalizedURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.apiClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	body := string(bodyBytes)

	// 2. Extract data-tralbum
	reTralbum := regexp.MustCompile(`data-tralbum="([^"]+)"`)
	matches := reTralbum.FindStringSubmatch(body)
	if len(matches) < 2 {
		return fmt.Errorf("data-tralbum not found on page")
	}

	decodedJSON := strings.ReplaceAll(matches[1], "&quot;", "\"")
	decodedJSON = strings.ReplaceAll(decodedJSON, "&amp;", "&")

	var td parsedTralbum
	if err := json.Unmarshal([]byte(decodedJSON), &td); err != nil {
		return fmt.Errorf("failed to parse tralbum json: %w", err)
	}

	itemTitle := td.Current.Title
	if itemTitle == "" {
		itemTitle = normalizedURL
	}
	progress(fmt.Sprintf("Processing: %s", itemTitle))

	// 3. Resolve Download Page URL
	downloadPageURL := td.FreeDownloadPage

	if downloadPageURL == "" {
		// Need email download
		progress("Requesting free download link via Bandcamp API...")
		dlURL, err := s.requestViaEmail(resp.Request.URL.String(), td, progress)
		if err != nil {
			return fmt.Errorf("email download error: %w", err)
		}
		downloadPageURL = dlURL
	}

	// 4. Parse download page
	progress("Accessing download page...")
	directDownloadURL, suggestedExt, err := s.extractDirectDownloadURL(downloadPageURL, format)
	if err != nil {
		return fmt.Errorf("failed to extract download URL: %w", err)
	}

	// 5. Download stream to file
	progress(fmt.Sprintf("Downloading file (%s)...", format))
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create download dir: %w", err)
	}

	return s.downloadStreamToFile(directDownloadURL, downloadDir, itemTitle, suggestedExt, progress)
}

func (s *DownloaderService) requestViaEmail(pageURL string, td parsedTralbum, progress ProgressCallback) (string, error) {
	// 1. Get temporary email address
	email, err := s.gmSvc.GetEmailAddress()
	if err != nil {
		log.Printf("Downloader: GuerrillaMail failed (%v), trying Mail.tm fallback...", err)
		email, err = s.tempEmailSvc.GenerateTempEmail()
		if err != nil {
			return "", fmt.Errorf("all temp email services failed: %w", err)
		}
	}
	progress(fmt.Sprintf("Using temporary email: %s", email))

	// 2. Submit form to /email_download
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("invalid page URL: %w", err)
	}
	endpoint := fmt.Sprintf("%s://%s/email_download", parsed.Scheme, parsed.Host)

	itemType := td.Current.Type
	if itemType == "" {
		itemType = td.ItemType
	}
	itemID := td.Current.ID
	if itemID == 0 {
		itemID = td.ID
	}

	formData := url.Values{
		"encoding_name": {"none"},
		"item_type":     {itemType},
		"item_id":       {fmt.Sprintf("%d", itemID)},
		"address":       {email},
		"country":       {"United States"},
		"postcode":      {"00000"},
	}

	postReq, err := http.NewRequest("POST", endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}
	postReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	postReq.Header.Set("Referer", pageURL)

	postResp, err := s.apiClient.Do(postReq)
	if err != nil {
		return "", fmt.Errorf("failed to send email request: %w", err)
	}
	defer postResp.Body.Close()

	respBytes, _ := io.ReadAll(postResp.Body)
	log.Printf("Downloader: Bandcamp email_download response: %s", string(respBytes))
	var postResult struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	json.Unmarshal(respBytes, &postResult)

	if !postResult.OK {
		if postResult.Error != "" {
			return "", fmt.Errorf("Bandcamp rejected email download: %s", postResult.Error)
		}
		return "", fmt.Errorf("Bandcamp rejected email download: %s", string(respBytes))
	}

	// 3. Poll for email
	progress("Waiting for Bandcamp email confirmation...")
	if s.gmSvc.EmailAddr == email {
		return s.gmSvc.WaitForDownloadLink(15, 5)
	}
	return s.tempEmailSvc.WaitForDownloadEmail(email, 15, 5)
}

func (s *DownloaderService) extractDirectDownloadURL(downloadPageURL string, requestedFormat string) (string, string, error) {
	req, err := http.NewRequest("GET", downloadPageURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.apiClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	body := string(bodyBytes)

	reBlob := regexp.MustCompile(`data-blob="([^"]+)"`)
	matches := reBlob.FindStringSubmatch(body)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("data-blob not found in download page")
	}

	decoded := strings.ReplaceAll(matches[1], "&quot;", "\"")
	decoded = strings.ReplaceAll(decoded, "&amp;", "&")

	var pageData pagedataBlob
	if err := json.Unmarshal([]byte(decoded), &pageData); err != nil {
		return "", "", fmt.Errorf("failed to parse download page data: %w", err)
	}

	if len(pageData.DigitalItems) == 0 {
		return "", "", fmt.Errorf("no digital items found on download page")
	}

	downloads := pageData.DigitalItems[0].Downloads
	fallbacks, ok := formatKeyMap[strings.ToLower(requestedFormat)]
	if !ok {
		fallbacks = []string{strings.ToLower(requestedFormat), "flac", "mp3-320"}
	}

	for _, fmtKey := range fallbacks {
		if dl, exists := downloads[fmtKey]; exists && dl.URL != "" {
			ext := fmtKey
			if ext == "mp3-320" || ext == "mp3-v0" {
				ext = "mp3"
			}
			return dl.URL, ext, nil
		}
	}

	// Fallback to any available format
	for fmtKey, dl := range downloads {
		if dl.URL != "" {
			return dl.URL, fmtKey, nil
		}
	}

	return "", "", fmt.Errorf("no suitable download formats found")
}

func (s *DownloaderService) downloadStreamToFile(downloadURL string, downloadDir string, title string, ext string, progress ProgressCallback) error {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.streamClient.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	// If expired or redirect needed
	if resp.StatusCode != http.StatusOK {
		statURL := strings.Replace(downloadURL, "/download/", "/statdownload/", 1)
		sReq, _ := http.NewRequest("GET", statURL, nil)
		sReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
		sResp, sErr := s.apiClient.Do(sReq)
		if sErr == nil {
			defer sResp.Body.Close()
			sBody, _ := io.ReadAll(sResp.Body)
			reRetry := regexp.MustCompile(`"retry_url":"([^"]+)"`)
			m := reRetry.FindStringSubmatch(string(sBody))
			if len(m) > 1 {
				retryURL := strings.ReplaceAll(m[1], `\/`, `/`)
				return s.downloadStreamToFile(retryURL, downloadDir, title, ext, progress)
			}
		}
		return fmt.Errorf("download server returned HTTP %d", resp.StatusCode)
	}

	// Determine file name
	filename := ""
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
			if fn, ok := params["filename"]; ok && fn != "" {
				filename = fn
			}
		}
	}

	if filename == "" {
		sanitizedTitle := sanitizeFilename(title)
		filename = fmt.Sprintf("%s.%s", sanitizedTitle, ext)
	}

	filePath := filepath.Join(downloadDir, filename)
	progress(fmt.Sprintf("Saving to: %s", filename))

	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	totalBytes := resp.ContentLength
	var downloadedBytes int64
	buf := make([]byte, 64*1024)
	lastLogTime := time.Now()

	for {
		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := outFile.Write(buf[:n]); wErr != nil {
				return fmt.Errorf("failed to write to file: %w", wErr)
			}
			downloadedBytes += int64(n)

			if time.Since(lastLogTime) > 2*time.Second {
				lastLogTime = time.Now()
				if totalBytes > 0 {
					pct := float64(downloadedBytes) / float64(totalBytes) * 100
					progress(fmt.Sprintf("Downloading %s: %.1f%% (%.1f / %.1f MB)", filename, pct, float64(downloadedBytes)/(1024*1024), float64(totalBytes)/(1024*1024)))
				} else {
					progress(fmt.Sprintf("Downloading %s: %.1f MB", filename, float64(downloadedBytes)/(1024*1024)))
				}
			}
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			return fmt.Errorf("download stream error: %w", rErr)
		}
	}

	progress(fmt.Sprintf("Download complete: %s", filename))
	log.Printf("Downloader: File downloaded successfully: %s (%d bytes)", filePath, downloadedBytes)

	// If zip file, optionally unzip if it's an album
	if strings.HasSuffix(strings.ToLower(filename), ".zip") {
		progress(fmt.Sprintf("Extracting %s...", filename))
		targetDir := filepath.Join(downloadDir, strings.TrimSuffix(filename, filepath.Ext(filename)))
		if err := unzipArchive(filePath, targetDir); err == nil {
			progress(fmt.Sprintf("Extracted album to %s", targetDir))
		}
	}

	return nil
}

func sanitizeFilename(name string) string {
	reg := regexp.MustCompile(`[\\/:*?"<>|]`)
	cleaned := reg.ReplaceAllString(name, "_")
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) > 100 {
		cleaned = cleaned[:100]
	}
	return cleaned
}

func unzipArchive(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
