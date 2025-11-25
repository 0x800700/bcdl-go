package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"bcdl-app/backend/models"
	"bcdl-app/backend/playwright"

	pw "github.com/playwright-community/playwright-go"
)

type ScannerService struct {
	pwService *playwright.Service
}

func NewScannerService(pwService *playwright.Service) *ScannerService {
	return &ScannerService{
		pwService: pwService,
	}
}

// ScanArtist scans a Bandcamp artist URL for albums
func (s *ScannerService) ScanArtist(ctx context.Context, url string, onAlbumFound func(models.Album)) ([]models.Album, error) {
	log.Printf("Scanner: Creating new page...")
	page, err := s.pwService.NewPage()
	if err != nil {
		log.Printf("Scanner: Failed to create page: %v", err)
		return nil, err
	}
	defer page.Close()

	// Navigate to artist page
	log.Printf("Scanner: Navigating to %s", url)
	if _, err := page.Goto(url, pw.PageGotoOptions{
		WaitUntil: pw.WaitUntilStateNetworkidle,
	}); err != nil {
		log.Printf("Scanner: Navigation failed: %v", err)
		return nil, fmt.Errorf("failed to navigate: %v", err)
	}
	log.Printf("Scanner: Navigation successful")

	// Wait for grid
	log.Printf("Scanner: Waiting for music grid...")
	grid := page.Locator("ol#music-grid")
	if err := grid.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}); err != nil {
		log.Printf("Scanner: Music grid not found: %v", err)
		return nil, fmt.Errorf("music grid not found: %v", err)
	}
	log.Printf("Scanner: Music grid found")

	// Extract all album data in one JavaScript call for performance
	log.Printf("Scanner: Extracting all album data via JavaScript...")
	result, err := page.Evaluate(`() => {
		const items = document.querySelectorAll('li.music-grid-item');
		return Array.from(items).map(item => {
			const titleEl = item.querySelector('.title');
			const artistEl = item.querySelector('.artist');
			const linkEl = item.querySelector('a');
			const coverEl = item.querySelector('img');
			const priceEl = item.querySelector('.price');
			
			// Handle lazy loading for cover image
			let coverUrl = '';
			if (coverEl) {
				coverUrl = coverEl.getAttribute('data-original') || coverEl.getAttribute('src');
			}

			return {
				title: titleEl ? titleEl.innerText.trim() : '',
				artist: artistEl ? artistEl.innerText.replace('by ', '').trim() : '',
				url: linkEl ? linkEl.getAttribute('href') : '',
				coverUrl: coverUrl,
				price: priceEl ? priceEl.innerText.trim() : ''
			};
		});
	}`)
	if err != nil {
		log.Printf("Scanner: Failed to extract data: %v", err)
		return nil, fmt.Errorf("failed to extract album data: %v", err)
	}

	// Convert result to []interface{}
	itemsData, ok := result.([]interface{})
	if !ok {
		log.Printf("Scanner: Unexpected result type: %T", result)
		return nil, fmt.Errorf("unexpected result type")
	}

	log.Printf("Scanner: Extracted %d albums from grid", len(itemsData))

	var albums []models.Album
	for i, itemData := range itemsData {
		// Check for cancellation
		select {
		case <-ctx.Done():
			log.Printf("Scanner: Scan cancelled by user")
			return albums, ctx.Err()
		default:
		}

		data, ok := itemData.(map[string]interface{})
		if !ok {
			continue
		}

		title, _ := data["title"].(string)
		artist, _ := data["artist"].(string)
		href, _ := data["url"].(string)
		coverURL, _ := data["coverUrl"].(string)
		// priceText is used as a fallback or initial guess
		// priceText, _ := data["price"].(string)

		// Handle relative URLs
		fullURL := href
		if !strings.HasPrefix(href, "http") {
			baseURL := strings.Split(url, "/music")[0]
			fullURL = baseURL + href
		}

		log.Printf("Scanner: Checking status for album %d/%d: %s", i+1, len(itemsData), title)

		// Visit album page to check true status (NYP/Free/Paid)
		// We reuse the same page for performance
		status := "paid"
		isFree := false
		isNYP := false

		if _, err := page.Goto(fullURL, pw.PageGotoOptions{
			WaitUntil: pw.WaitUntilStateDomcontentloaded, // Faster than networkidle
		}); err == nil {
			// Check for "name your price" or "Free Download"
			// Using Evaluate for speed
			checkResult, err := page.Evaluate(`() => {
				const buyHeader = document.querySelector('h4.ft.compound-button');
				if (!buyHeader) return 'unavailable';
				
				const text = buyHeader.innerText.toLowerCase();
				if (text.includes('name your price')) return 'nyp';
				if (text.includes('free download')) return 'free';
				
				const buyBtn = buyHeader.querySelector('button.download-link');
				if (buyBtn) {
					const btnText = buyBtn.innerText.toLowerCase();
					if (btnText.includes('name your price')) return 'nyp';
					if (btnText.includes('free')) return 'free';
				}
				
				return 'paid';
			}`)

			if err == nil {
				statusStr, _ := checkResult.(string)
				if statusStr == "nyp" {
					isNYP = true
					status = "nyp"
				} else if statusStr == "free" {
					isFree = true
					status = "free"
				} else if statusStr == "paid" {
					status = "paid"
				}
			}
		} else {
			log.Printf("Scanner: Failed to visit album page: %v", err)
		}

		album := models.Album{
			Title:    title,
			Artist:   artist,
			CoverURL: coverURL,
			URL:      fullURL,
			IsFree:   isFree,
			IsNYP:    isNYP,
			Price:    "", // Price text is less relevant now that we have status
			Status:   status,
		}

		albums = append(albums, album)

		// Emit event for dynamic UI updates
		if onAlbumFound != nil {
			onAlbumFound(album)
		}
	}

	log.Printf("Scanner: Finished processing all items, returning %d albums", len(albums))
	return albums, nil
}
