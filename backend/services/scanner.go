package services

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"bcdl-app/backend/models"

	"github.com/PuerkitoBio/goquery"
)

type ScannerService struct {
}

func NewScannerService() *ScannerService {
	return &ScannerService{}
}

// ScanArtist scans a Bandcamp artist URL for albums
func (s *ScannerService) ScanArtist(ctx context.Context, urlStr string, onAlbumFound func(models.Album)) ([]models.Album, error) {
	log.Printf("Scanner: Fetching artist page via HTTP: %s", urlStr)
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artist page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("artist page returned status code: %d", res.StatusCode)
	}
	
	resolvedURL := res.Request.URL.String()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	var rawAlbums []models.Album

	defaultArtist := strings.TrimSpace(doc.Find("#band-name-location .title").Text())
	if defaultArtist == "" {
		defaultArtist = strings.TrimSpace(doc.Find("meta[property='og:site_name']").AttrOr("content", ""))
	}

	log.Printf("Scanner: Parsing music grid...")
	doc.Find("ol#music-grid li.music-grid-item, ul#music-grid li.music-grid-item").Each(func(i int, sel *goquery.Selection) {
		var title string
		sel.Find(".title").Contents().Each(func(i int, s *goquery.Selection) {
			if goquery.NodeName(s) == "#text" {
				title += s.Text()
			}
		})
		title = strings.TrimSpace(title)
		
		artist := strings.TrimSpace(sel.Find(".artist-override").Text())
		if artist == "" {
			artist = strings.TrimSpace(sel.Find(".artist").Text())
			artist = strings.TrimPrefix(artist, "by ")
			artist = strings.TrimSpace(artist)
		}
		if artist == "" {
			artist = defaultArtist
		}

		href, exists := sel.Find("a").Attr("href")
		if !exists {
			return
		}

		fullURL := href
		if !strings.HasPrefix(href, "http") {
			baseURL := strings.TrimRight(strings.Split(resolvedURL, "/music")[0], "/")
			if !strings.HasPrefix(href, "/") {
				fullURL = baseURL + "/" + href
			} else {
				fullURL = baseURL + href
			}
		}

		coverURL := ""
		imgSel := sel.Find("img")
		if lazySrc, ok := imgSel.Attr("data-original"); ok && lazySrc != "" {
			coverURL = lazySrc
		} else if src, ok := imgSel.Attr("src"); ok && src != "" {
			coverURL = src
		}

		rawAlbums = append(rawAlbums, models.Album{
			Title:    title,
			Artist:   artist,
			CoverURL: coverURL,
			URL:      fullURL,
		})
	})

	// Also parse data-client-items from the music grid (Bandcamp stores remaining items here beyond the first 16)
	grid := doc.Find("ol#music-grid, ul#music-grid")
	if clientItems, exists := grid.Attr("data-client-items"); exists && clientItems != "" {
		unescaped := html.UnescapeString(clientItems)
		var items []struct {
			Title    string `json:"title"`
			PageURL  string `json:"page_url"`
			ArtID    int64  `json:"art_id"`
			Type     string `json:"type"`
			BandID   int64  `json:"band_id"`
			ID       int64  `json:"id"`
			Filtered bool   `json:"filtered"`
		}
		if err := json.Unmarshal([]byte(unescaped), &items); err == nil {
			log.Printf("Scanner: Found %d additional items in data-client-items", len(items))
			baseURL := strings.TrimRight(strings.Split(resolvedURL, "/music")[0], "/")
			for _, item := range items {
				if item.Filtered || item.PageURL == "" {
					continue
				}
				fullURL := item.PageURL
				if !strings.HasPrefix(fullURL, "http") {
					if !strings.HasPrefix(fullURL, "/") {
						fullURL = baseURL + "/" + fullURL
					} else {
						fullURL = baseURL + fullURL
					}
				}

				coverURL := ""
				if item.ArtID > 0 {
					coverURL = fmt.Sprintf("https://f4.bcbits.com/img/a%d_9.jpg", item.ArtID)
				}

				// Avoid duplicates
				alreadyExists := false
				for _, existing := range rawAlbums {
					if existing.URL == fullURL {
						alreadyExists = true
						break
					}
				}

				if !alreadyExists {
					rawAlbums = append(rawAlbums, models.Album{
						Title:    item.Title,
						Artist:   defaultArtist,
						CoverURL: coverURL,
						URL:      fullURL,
					})
				}
			}
		}
	}

	log.Printf("Scanner: Extracted %d total releases. Checking price status concurrently...", len(rawAlbums))

	var albums []models.Album
	var mu sync.Mutex
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for i, rawAlbum := range rawAlbums {
		select {
		case <-ctx.Done():
			log.Printf("Scanner: Scan cancelled by user")
			return albums, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(index int, album models.Album) {
			defer wg.Done()
			defer func() { <-sem }()

			status, isFree, isNYP := checkAlbumStatus(ctx, client, album.URL)
			album.Status = status
			album.IsFree = isFree
			album.IsNYP = isNYP

			log.Printf("Scanner: Status for album %d/%d (%s): %s", index+1, len(rawAlbums), album.Title, status)

			mu.Lock()
			albums = append(albums, album)
			if onAlbumFound != nil {
				onAlbumFound(album)
			}
			mu.Unlock()

		}(i, rawAlbum)
	}

	wg.Wait()
	
	log.Printf("Scanner: Finished processing all items, returning %d albums", len(albums))
	return albums, nil
}

func checkAlbumStatus(ctx context.Context, client *http.Client, urlStr string) (string, bool, bool) {
	status := "paid"
	isFree := false
	isNYP := false

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return status, isFree, isNYP
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	res, err := client.Do(req)
	if err != nil {
		return status, isFree, isNYP
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return status, isFree, isNYP
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return status, isFree, isNYP
	}

	buyHeader := doc.Find("h4.ft.compound-button")
	if buyHeader.Length() == 0 {
		return "unavailable", isFree, isNYP
	}

	text := strings.ToLower(buyHeader.Text())
	if strings.Contains(text, "name your price") {
		return "nyp", false, true
	}
	if strings.Contains(text, "free download") {
		return "free", true, false
	}

	buyBtn := buyHeader.Find("button.download-link, a.download-link")
	if buyBtn.Length() > 0 {
		btnText := strings.ToLower(buyBtn.Text())
		if strings.Contains(btnText, "name your price") {
			return "nyp", false, true
		}
		if strings.Contains(btnText, "free") {
			return "free", true, false
		}
	}

	return status, isFree, isNYP
}
