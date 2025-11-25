package services

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"bcdl-app/backend/playwright"

	pw "github.com/playwright-community/playwright-go"
)

type DownloaderService struct {
	pwService    *playwright.Service
	tempEmailSvc *TempEmailService
}

func NewDownloaderService(pwService *playwright.Service) *DownloaderService {
	return &DownloaderService{
		pwService:    pwService,
		tempEmailSvc: NewTempEmailService(),
	}
}

// ProgressCallback is a function that receives progress updates
type ProgressCallback func(message string)

func (s *DownloaderService) DownloadAlbum(url string, downloadDir string, format string, progress ProgressCallback) error {
	log.Printf("Downloader: Starting download for: %s", url)
	progress(fmt.Sprintf("Starting download for: %s", url))

	page, err := s.pwService.NewPage()
	if err != nil {
		return err
	}
	defer page.Close()

	// Navigate to album page
	if _, err := page.Goto(url, pw.PageGotoOptions{
		WaitUntil: pw.WaitUntilStateDomcontentloaded, // Relaxed from Networkidle
	}); err != nil {
		return fmt.Errorf("failed to navigate: %v", err)
	}

	// Handle Cookie Banner (Critical for interaction)
	// We use a more generic selector to catch the banner container or button
	log.Printf("Downloader: Checking for cookie banner...")
	cookieBtn := page.Locator("#onetrust-accept-btn-handler").First() // Try ID first (common on Bandcamp)
	if count, _ := cookieBtn.Count(); count == 0 {
		// Fallback to text search if ID not found
		cookieBtn = page.Locator("button").Filter(pw.LocatorFilterOptions{HasText: "Accept all"}).First()
	}

	if err := cookieBtn.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(5000)}); err == nil {
		progress("Found cookie banner, clicking 'Accept all'...")
		log.Printf("Downloader: Clicking cookie button...")
		if err := cookieBtn.Click(); err != nil {
			log.Printf("Downloader: Failed to click cookie button: %v", err)
		}

		// Wait for banner to disappear/page to update
		time.Sleep(1 * time.Second)
	} else {
		log.Printf("Downloader: Cookie banner not found (timeout)")
	}

	// Get title
	titleEl := page.Locator("h2.trackTitle").First()
	if err := titleEl.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(10000)}); err != nil {
		log.Printf("Downloader: Title element not found: %v", err)
	}
	title, _ := titleEl.InnerText()
	title = strings.TrimSpace(title)
	log.Printf("Downloader: Processing album: %s", title)
	progress(fmt.Sprintf("Processing album: %s", title))

	// 1. Direct link check (optimization)
	log.Printf("Downloader: Checking for direct download link...")
	progress("Checking for direct download link...")
	tralbumData, err := page.Locator("script[data-tralbum]").GetAttribute("data-tralbum")
	if err == nil && tralbumData != "" {
		// Simple string check to avoid full JSON parsing if possible, or use Evaluate for robust check
		isFree, _ := page.Evaluate(`() => {
			try {
				const data = JSON.parse(document.querySelector("script[data-tralbum]").getAttribute("data-tralbum"));
				return data && data.freeDownloadPage ? data.freeDownloadPage : null;
			} catch (e) { return null; }
		}`)

		if freePage, ok := isFree.(string); ok && freePage != "" {
			progress("Found direct download link, skipping payment flow...")
			if _, err := page.Goto(freePage); err != nil {
				return fmt.Errorf("failed to navigate to free download page: %v", err)
			}
			return s.handleDownloadPage(page, downloadDir, format, progress)
		}
	}
	progress("No direct link found, proceeding with buy button...")

	// 2. Buy/Free button interaction
	// Python uses regex: Buy|Free|Download. We'll use a broader selector and check visibility.
	log.Printf("Downloader: Looking for buy/download button...")
	progress("Looking for buy/download button...")

	// Try primary selector (works for most albums) - removed "button" tag constraint to support <a> tags
	buyBtn := page.Locator("h4.ft.compound-button .download-link").First()

	// Check if button exists and is visible
	if err := buyBtn.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(3000)}); err != nil {
		log.Printf("Downloader: Primary buy button selector failed, trying fallback...")

		// Fallback: Look for "Buy Digital Album" or "name your price" text
		buyBtn = page.Locator("text=Buy Digital Album").First()
		if err := buyBtn.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(3000)}); err != nil {
			buyBtn = page.Locator("text=name your price").First()
			if err := buyBtn.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(3000)}); err != nil {
				log.Printf("Downloader: No download button found (all selectors failed)")
				progress("No download button found")
				return fmt.Errorf("no download button found")
			}
		}
	}
	log.Printf("Downloader: Found buy/download button")
	progress("Found buy/download button")

	progress("Clicking buy/download button...")
	if err := buyBtn.Click(pw.LocatorClickOptions{Force: pw.Bool(true)}); err != nil {
		return fmt.Errorf("failed to click buy button: %v", err)
	}
	progress("Buy button clicked, checking for price input...")

	// 3. Price input (Name Your Price)
	priceInput := page.Locator("input#userPrice")
	log.Printf("Downloader: Waiting for price input field...")
	progress("Waiting for price input field...")
	if err := priceInput.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(5000)}); err == nil {
		log.Printf("Downloader: Price input found, setting to 0...")
		progress("Price input found, setting to 0...")
		if err := priceInput.Fill("0"); err != nil {
			return fmt.Errorf("failed to set price: %v", err)
		}

		// Click "download to your computer" link
		// This link appears after typing 0
		progress("Looking for 'download to your computer' link...")
		downloadLink := page.Locator("a.download-panel-free-download-link")
		if err := downloadLink.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(5000)}); err == nil {
			progress("Found download link, clicking...")
			if err := downloadLink.Click(pw.LocatorClickOptions{Force: pw.Bool(true)}); err != nil {
				return fmt.Errorf("failed to click free download link: %v", err)
			}

			// Wait for page to load
			progress("Waiting for page to load after clicking download link...")
			page.WaitForLoadState(pw.PageWaitForLoadStateOptions{
				State: pw.LoadStateNetworkidle,
			})

			// Check if email form appeared FIRST (URL might not change!)
			currentURL := page.URL()
			log.Printf("Downloader: Current URL after click: %s", currentURL)
			progress(fmt.Sprintf("Current URL after click: %s", currentURL))

			emailInputCount, _ := page.Locator("input#fan_email_address").Count()
			log.Printf("Downloader: Email input count: %d", emailInputCount)

			if emailInputCount > 0 {
				progress(fmt.Sprintf("Email form detected (%d inputs found)", emailInputCount))
				log.Printf("Downloader: Email form detected, starting temp email flow")
				return s.handleEmailFlow(page, downloadDir, format, progress)
			} else if strings.Contains(currentURL, "download") {
				progress("URL contains 'download' - proceeding to download page")
				log.Printf("Downloader: URL contains 'download', proceeding to download page")
				// Continue to download page handling
			} else {
				progress("No download page or email form detected - unexpected state")
				log.Printf("Downloader: Unexpected state - no email form and URL doesn't contain 'download'")
			}
		} else {
			// Check if email form is visible (alternative flow)
			if count, _ := page.Locator("input#fan_email_address").Count(); count > 0 {
				progress("Email required - using temp email flow...")
				return s.handleEmailFlow(page, downloadDir, format, progress)
			}
			return fmt.Errorf("free download link not found after setting price")
		}
	}

	// 4. Handle actual download page
	return s.handleDownloadPage(page, downloadDir, format, progress)
}

// handleEmailFlow handles the temp email verification flow
func (s *DownloaderService) handleEmailFlow(page pw.Page, downloadDir string, format string, progress ProgressCallback) error {
	// Generate temp email
	tempEmail, err := s.tempEmailSvc.GenerateTempEmail()
	if err != nil {
		return fmt.Errorf("failed to generate temp email: %v", err)
	}
	progress(fmt.Sprintf("Generated temp email: %s", tempEmail))

	// Fill email form
	emailInput := page.Locator("input#fan_email_address")
	if err := emailInput.Fill(tempEmail); err != nil {
		return fmt.Errorf("failed to fill email: %v", err)
	}

	// Fill ZIP code (using a generic US ZIP)
	zipInput := page.Locator("input[name='postcode'], input.postcode")
	if err := zipInput.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(3000)}); err == nil {
		progress("Filling ZIP code...")
		zipInput.Fill("10001")
	}

	// Click OK button
	okBtn := page.Locator("button").Filter(pw.LocatorFilterOptions{HasText: "OK"}).First()
	if err := okBtn.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(5000)}); err != nil {
		return fmt.Errorf("OK button not found: %v", err)
	}

	progress("Submitting email form...")
	if err := okBtn.Click(pw.LocatorClickOptions{Force: pw.Bool(true)}); err != nil {
		return fmt.Errorf("failed to click OK button: %v", err)
	}

	// Wait for download email (max 120 seconds, check every 5 seconds)
	downloadLink, err := s.tempEmailSvc.WaitForDownloadEmail(tempEmail, 24, 5)
	if err != nil {
		return fmt.Errorf("failed to receive download email: %v", err)
	}

	progress(fmt.Sprintf("Received download link: %s", downloadLink))

	// Navigate to download link
	if _, err := page.Goto(downloadLink, pw.PageGotoOptions{
		WaitUntil: pw.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("failed to navigate to download link: %v", err)
	}

	// Continue with normal download flow
	return s.handleDownloadPage(page, downloadDir, format, progress)
}

func (s *DownloaderService) handleDownloadPage(page pw.Page, downloadDir string, format string, progress ProgressCallback) error {
	progress("Waiting for download page...")

	// Wait for format selector
	formatDropdown := page.Locator("#format-type, .format-type, .formats").First()
	if err := formatDropdown.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(20000)}); err != nil {
		return fmt.Errorf("format selector not found (timeout): %v", err)
	}

	// Select format
	tagName, _ := formatDropdown.Evaluate("el => el.tagName", nil)
	if tagName == "SELECT" {
		if _, err := formatDropdown.SelectOption(pw.SelectOptionValues{
			Values: pw.StringSlice(strings.ToLower(format)),
		}); err != nil {
			// Fallback to MP3 320 if FLAC not found or error
			progress("Requested format not found, trying MP3 320...")
			formatDropdown.SelectOption(pw.SelectOptionValues{
				Values: pw.StringSlice("mp3-320"),
			})
		}
	} else {
		// Custom dropdown
		formatDropdown.Click()
		page.Locator("li").Filter(pw.LocatorFilterOptions{HasText: format}).Click()
	}
	progress(fmt.Sprintf("Selected format: %s", format))

	// Find Download button
	downloadBtn := page.Locator(".download-item-container a").Filter(pw.LocatorFilterOptions{HasText: "Download"}).First()
	progress("Preparing download...")
	if err := downloadBtn.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(60000)}); err != nil {
		return fmt.Errorf("download button timeout: %v", err)
	}

	// Handle download
	download, err := page.ExpectDownload(func() error {
		return downloadBtn.Click()
	})
	if err != nil {
		return fmt.Errorf("download failed to start: %v", err)
	}

	// Save file
	suggestedFilename := download.SuggestedFilename()
	savePath := filepath.Join(downloadDir, suggestedFilename)

	progress(fmt.Sprintf("Saving to: %s", savePath))
	if err := download.SaveAs(savePath); err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	progress("Download complete!")
	return nil
}
