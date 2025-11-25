package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type TempEmailService struct {
	client    *http.Client
	Token     string
	AccountID string
	Email     string
}

// Mail.tm API structures
type DomainResponse struct {
	HydraMember []struct {
		Domain string `json:"domain"`
	} `json:"hydra:member"`
}

type AccountRequest struct {
	Address  string `json:"address"`
	Password string `json:"password"`
}

type AccountResponse struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type EmailMessage struct {
	ID   string `json:"id"` // Mail.tm uses string IDs
	From struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	} `json:"from"`
	Subject string `json:"subject"`
	Intro   string `json:"intro"`
}

type EmailBody struct {
	ID   string `json:"id"`
	From struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	} `json:"from"`
	Subject string   `json:"subject"`
	Html    []string `json:"html"`
	Text    string   `json:"text"`
}

func NewTempEmailService() *TempEmailService {
	return &TempEmailService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateTempEmail creates a new temporary email address using Mail.tm
func (s *TempEmailService) GenerateTempEmail() (string, error) {
	// 1. Get Domains
	domain, err := s.getDomain()
	if err != nil {
		return "", fmt.Errorf("failed to get domain: %v", err)
	}

	// 2. Generate Credentials
	username := fmt.Sprintf("user%d", time.Now().UnixNano())
	password := fmt.Sprintf("Pwd%d!", time.Now().UnixNano())
	address := fmt.Sprintf("%s@%s", username, domain)

	// 3. Create Account
	if err := s.createAccount(address, password); err != nil {
		return "", fmt.Errorf("failed to create account: %v", err)
	}

	// 4. Get Token
	token, err := s.getToken(address, password)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}

	s.Email = address
	s.Token = token
	log.Printf("TempEmail: Generated email: %s", s.Email)

	return s.Email, nil
}

func (s *TempEmailService) getDomain() (string, error) {
	resp, err := s.client.Get("https://api.mail.tm/domains")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result DomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.HydraMember) == 0 {
		return "", fmt.Errorf("no domains available")
	}

	return result.HydraMember[0].Domain, nil
}

func (s *TempEmailService) createAccount(address, password string) error {
	reqBody, _ := json.Marshal(AccountRequest{
		Address:  address,
		Password: password,
	})

	resp, err := s.client.Post("https://api.mail.tm/accounts", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result AccountResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	s.AccountID = result.ID
	return nil
}

func (s *TempEmailService) getToken(address, password string) (string, error) {
	reqBody, _ := json.Marshal(AccountRequest{
		Address:  address,
		Password: password,
	})

	resp, err := s.client.Post("https://api.mail.tm/token", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

// CheckInbox polls the inbox for new messages
func (s *TempEmailService) CheckInbox(email string) ([]EmailMessage, error) {
	req, err := http.NewRequest("GET", "https://api.mail.tm/messages", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check inbox: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d", resp.StatusCode)
	}

	var result struct {
		HydraMember []EmailMessage `json:"hydra:member"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse messages: %v", err)
	}

	return result.HydraMember, nil
}

// ReadMessage retrieves the full content of a message
func (s *TempEmailService) ReadMessage(email string, messageID string) (*EmailBody, error) {
	url := fmt.Sprintf("https://api.mail.tm/messages/%s", messageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d", resp.StatusCode)
	}

	var emailBody EmailBody
	if err := json.NewDecoder(resp.Body).Decode(&emailBody); err != nil {
		return nil, fmt.Errorf("failed to parse message: %v", err)
	}

	return &emailBody, nil
}

// ExtractDownloadLink extracts the Bandcamp download link from email body
func (s *TempEmailService) ExtractDownloadLink(emailBody string) (string, error) {
	// Bandcamp sends links like: https://bandcamp.com/download?...
	// or https://[artist].bandcamp.com/download?...
	// Regex updated to handle both cases (optional subdomain)
	re := regexp.MustCompile(`https?://[^"'\s<>]*bandcamp\.com/download[^"'\s<>]*`)
	matches := re.FindStringSubmatch(emailBody)

	if len(matches) == 0 {
		// Log truncated body for debugging
		debugBody := emailBody
		if len(debugBody) > 500 {
			debugBody = debugBody[:500] + "..."
		}
		log.Printf("TempEmail: No link found in body snippet: %s", debugBody)
		return "", fmt.Errorf("no download link found in email")
	}

	link := matches[0]
	// Clean up any HTML entities or trailing characters
	link = strings.TrimRight(link, "\"'<>")

	log.Printf("TempEmail: Extracted download link: %s", link)
	return link, nil
}

// WaitForDownloadEmail polls the inbox until a Bandcamp email arrives or timeout
func (s *TempEmailService) WaitForDownloadEmail(email string, maxAttempts int, intervalSeconds int) (string, error) {
	log.Printf("TempEmail: Waiting for download email at %s...", email)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("TempEmail: Polling inbox... (attempt %d/%d)", attempt, maxAttempts)

		messages, err := s.CheckInbox(email)
		if err != nil {
			log.Printf("TempEmail: Failed to check inbox: %v", err)
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
			continue
		}

		// Look for Bandcamp email
		for _, msg := range messages {
			log.Printf("TempEmail: Checking message: From='%s' (%s), Subject='%s'", msg.From.Name, msg.From.Address, msg.Subject)

			// Check sender address OR name OR subject
			isBandcamp := strings.Contains(strings.ToLower(msg.From.Address), "bandcamp.com") ||
				strings.Contains(strings.ToLower(msg.From.Name), "bandcamp")

			isDownload := strings.Contains(strings.ToLower(msg.Subject), "download")

			if isBandcamp || isDownload {
				log.Printf("TempEmail: Match found! From: %s, Subject: %s", msg.From.Address, msg.Subject)

				// Read full message
				emailBody, err := s.ReadMessage(email, msg.ID)
				if err != nil {
					log.Printf("TempEmail: Failed to read message: %v", err)
					continue
				}

				// Extract download link from HTML (preferred) or Text
				var link string
				if len(emailBody.Html) > 0 {
					link, err = s.ExtractDownloadLink(emailBody.Html[0])
				}
				if link == "" || err != nil {
					link, err = s.ExtractDownloadLink(emailBody.Text)
				}

				if err != nil {
					log.Printf("TempEmail: Failed to extract link: %v", err)
					continue
				}

				return link, nil
			}
		}

		if attempt < maxAttempts {
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
		}
	}

	return "", fmt.Errorf("timeout waiting for download email after %d attempts", maxAttempts)
}
