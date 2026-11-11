package requestexternal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"monorepo/bin-message-manager/models/telnyx"
	"net/http"

	"github.com/pkg/errors"
)

// TelnyxSendMessage sends request to the telnyx to send the message.
func (h *requestExternal) TelnyxSendMessage(ctx context.Context, source string, destination string, text string) (*telnyx.MessageResponse, error) {
	token, err := h.authtokenTelnyx(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: telnyx: %w", ErrTokenUnavailable, err)
	}

	uri := "https://api.telnyx.com/v2/messages"

	payload := struct {
		From      string   `json:"from"`
		To        string   `json:"to"`
		Text      string   `json:"text"`
		MediaURLs []string `json:"media_urls,omitempty"`
	}{
		From: source,
		To:   destination,
		Text: text,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "error marshalling JSON")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "error creating request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token) // Use Bearer token authentication

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error making request")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// the raw body carries the numbers, so only the error codes and descriptions are kept.
		return nil, &ProviderError{Provider: providerTelnyx, StatusCode: resp.StatusCode, Detail: providerErrorDetail(resp.Body)}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading response body")
	}

	var messageResponse telnyx.MessageResponse
	if errUnmarshal := json.Unmarshal(bodyBytes, &messageResponse); errUnmarshal != nil {
		return nil, errors.Wrapf(errUnmarshal, "error unmarshalling JSON")
	}

	return &messageResponse, nil
}
