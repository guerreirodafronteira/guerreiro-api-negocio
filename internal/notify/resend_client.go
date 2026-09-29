package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ResendClient struct {
	apiKey    string
	fromEmail string
}

func NewResendClient(apiKey, fromEmail string) *ResendClient {
	return &ResendClient{apiKey: apiKey, fromEmail: fromEmail}
}

type resendEmailPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (c *ResendClient) SendEmail(to, subject, html string) error {
	payload := resendEmailPayload{
		From:    c.fromEmail,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload do email: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("erro ao montar requisição: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao chamar API do Resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend retornou status %d", resp.StatusCode)
	}

	return nil
}