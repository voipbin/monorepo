package whatsapphandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
)

// GraphAPIBase is exported so tests can override it with a local httptest server.
var GraphAPIBase = "https://graph.facebook.com"

const graphAPIVersion = "v19.0"

// Send sends a text message to the WhatsApp user via Meta Cloud API.
// It returns the wamid (WhatsApp message ID) assigned by Meta.
func (h *whatsappHandler) Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "whatsapphandler.Send",
		"conversation_id": cv.ID,
	})

	var pd account.WhatsAppProviderData
	if err := json.Unmarshal(ac.ProviderData, &pd); err != nil {
		return "", fmt.Errorf("whatsapphandler.Send: invalid provider_data: %w", err)
	}

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                cv.DialogID,
		"type":              "text",
		"text":              map[string]string{"body": text},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("whatsapphandler.Send: marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s/messages", GraphAPIBase, graphAPIVersion, pd.PhoneNumberID)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("whatsapphandler.Send: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+ac.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("whatsapphandler.Send: http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("whatsapphandler.Send: API returned %d", resp.StatusCode)
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("whatsapphandler.Send: decode response: %w", err)
	}
	if len(result.Messages) == 0 {
		return "", fmt.Errorf("whatsapphandler.Send: no messages in response")
	}

	log.Debugf("Sent WhatsApp message. wamid: %s", result.Messages[0].ID)
	return result.Messages[0].ID, nil
}
