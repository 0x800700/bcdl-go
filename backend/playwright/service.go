package playwright

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"
)

type Service struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Init() error {
	var err error

	// Check for local "browsers" folder (Portable Mode)
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	localBrowsersPath := filepath.Join(exeDir, "browsers")
	portableMode := false
	if _, err := os.Stat(localBrowsersPath); err == nil {
		log.Printf("Found local browsers folder: %s (Portable Mode)", localBrowsersPath)
		os.Setenv("PLAYWRIGHT_BROWSERS_PATH", localBrowsersPath)
		portableMode = true
	}

	// Install driver and browsers only if NOT in portable mode
	if !portableMode {
		log.Println("Installing Playwright browsers...")
		if err := playwright.Install(); err != nil {
			return fmt.Errorf("could not install playwright browsers: %v", err)
		}
	} else {
		log.Println("Using bundled browsers (Portable Mode)")
	}

	s.pw, err = playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %v", err)
	}

	// Launch options
	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // Headless mode for production
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
		},
	}

	// Check for debug mode via env var
	if os.Getenv("DEBUG") == "true" {
		launchOptions.Headless = playwright.Bool(false)
	}

	s.browser, err = s.pw.Chromium.Launch(launchOptions)
	if err != nil {
		return fmt.Errorf("could not launch browser: %v", err)
	}

	log.Println("Playwright initialized successfully")
	return nil
}

func (s *Service) NewPage() (playwright.Page, error) {
	if s.browser == nil {
		return nil, fmt.Errorf("browser not initialized")
	}

	// Create context with user agent to avoid detection
	context, err := s.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent:       playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		AcceptDownloads: playwright.Bool(true), // Added AcceptDownloads
	})
	if err != nil {
		return nil, fmt.Errorf("could not create context: %v", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %v", err)
	}

	return page, nil
}

func (s *Service) Close() {
	if s.browser != nil {
		s.browser.Close()
	}
	if s.pw != nil {
		s.pw.Stop()
	}
}
