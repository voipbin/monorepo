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
	req.Header.Set("Authorization", "Bearer "+h.authtokenTelnyx) // Use Bearer token authentication

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error making request")
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading response body")
	}
	bodyString := string(bodyBytes)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status code %d and body: %s", resp.StatusCode, bodyString)
	}

	var messageResponse telnyx.MessageResponse
	if errUnmarshal := json.Unmarshal(bodyBytes, &messageResponse); errUnmarshal != nil {
		return nil, errors.Wrapf(errUnmarshal, "error unmarshalling JSON")
	}

	return &messageResponse, nil
}
