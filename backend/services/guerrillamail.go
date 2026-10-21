package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type GuerrillaMailService struct {
	client    *http.Client
	EmailAddr string
	SidToken  string
}

type gmAddressResp struct {
	EmailAddr string `json:"email_addr"`
	SidToken  string `json:"sid_token"`
}

type gmEmailList struct {
	List []struct {
		MailID      interface{} `json:"mail_id"`
		MailFrom    string      `json:"mail_from"`
		MailSubject string      `json:"mail_subject"`
	} `json:"list"`
}

type gmEmailBody struct {
	MailBody string `json:"mail_body"`
}

func NewGuerrillaMailService() *GuerrillaMailService {
	return &GuerrillaMailService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetEmailAddress initializes session and returns disposable address
func (s *GuerrillaMailService) GetEmailAddress() (string, error) {
	resp, err := s.client.Get("https://api.guerrillamail.com/ajax.php?f=get_email_address")
	if err != nil {
		return "", fmt.Errorf("failed to get guerrillamail address: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var data gmAddressResp
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("failed to decode json: %w", err)
	}

	s.EmailAddr = data.EmailAddr
	s.SidToken = data.SidToken
	log.Printf("GuerrillaMail: Generated email %s", s.EmailAddr)
	return s.EmailAddr, nil
}

// WaitForDownloadLink polls for an email from Bandcamp containing download link
func (s *GuerrillaMailService) WaitForDownloadLink(maxAttempts int, intervalSeconds int) (string, error) {
	if s.SidToken == "" {
		return "", fmt.Errorf("guerrillamail session not initialized")
	}

	reLink := regexp.MustCompile(`https?://[^"'\s]*bandcamp\.com/download\?[^"'\s<]*`)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		time.Sleep(time.Duration(intervalSeconds) * time.Second)

		checkURL := fmt.Sprintf("https://api.guerrillamail.com/ajax.php?f=check_email&seq=0&sid_token=%s", s.SidToken)
		resp, err := s.client.Get(checkURL)
		if err != nil {
			log.Printf("GuerrillaMail: Check failed (attempt %d): %v", attempt, err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var list gmEmailList
		if err := json.Unmarshal(body, &list); err != nil {
			log.Printf("GuerrillaMail: Failed to decode check_email JSON: %v", err)
			continue
		}

		log.Printf("GuerrillaMail: Polling (attempt %d/%d, found %d emails)", attempt, maxAttempts, len(list.List))

		for _, msg := range list.List {
			if strings.Contains(strings.ToLower(msg.MailFrom), "bandcamp") || strings.Contains(strings.ToLower(msg.MailSubject), "download") {
				log.Printf("GuerrillaMail: Received email from %s: %s", msg.MailFrom, msg.MailSubject)

				mailID := fmt.Sprintf("%v", msg.MailID)
				fetchURL := fmt.Sprintf("https://api.guerrillamail.com/ajax.php?f=fetch_email&email_id=%s&sid_token=%s", mailID, s.SidToken)
				bResp, err := s.client.Get(fetchURL)
				if err != nil {
					continue
				}
				bBytes, _ := io.ReadAll(bResp.Body)
				bResp.Body.Close()

				var b gmEmailBody
				if err := json.Unmarshal(bBytes, &b); err != nil {
					continue
				}

				match := reLink.FindString(b.MailBody)
				if match != "" {
					return match, nil
				}
			}
		}
	}

	return "", fmt.Errorf("timed out waiting for Bandcamp email")
}
