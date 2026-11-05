package requestexternal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"monorepo/bin-message-manager/models/messagebird"
	"monorepo/bin-message-manager/models/telnyx"
)

const (
	providerTelnyx      = "telnyx"
	providerMessagebird = "messagebird"

	telnyxMessagesURI      = "https://api.telnyx.com/v2/messages"
	messagebirdMMSURI      = "https://rest.messagebird.com/mms"
	statusRequestTimeout   = http.StatusRequestTimeout // the one 4xx status that does not prove the provider refused the order
	httpStatusClientErrMin = 400
	httpStatusClientErrMax = 499
)

// ProviderError is returned when a provider answered an MMS request with a non 2xx status.
// It carries the provider name and the status only: the response body is dropped on purpose,
// because it can echo the media URLs, which must not reach the logs.
type ProviderError struct {
	Provider   string
	StatusCode int
}

// Error implements the error interface.
func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s refused the request. status_code: %d", e.Provider, e.StatusCode)
}

// IsDefinitiveRejection returns true if the error proves that the provider refused the request,
// so nothing was queued at the provider. Every status from 400 to 499 except 408 counts. A 408,
// a 5xx, a timeout, a transport error and an unreadable answer do not: the message may exist at
// the provider, so the caller must not send it again through another provider.
func IsDefinitiveRejection(err error) bool {
	var pe *ProviderError
	if !errors.As(err, &pe) {
		return false
	}

	return pe.StatusCode >= httpStatusClientErrMin && pe.StatusCode <= httpStatusClientErrMax && pe.StatusCode != statusRequestTimeout
}

// TelnyxSendMMS sends one MMS to the telnyx.
func (h *requestExternal) TelnyxSendMMS(ctx context.Context, source string, destination string, text string, mediaURLs []string) (*telnyx.MessageResponse, error) {
	payload := struct {
		From      string   `json:"from"`
		To        string   `json:"to"`
		Text      string   `json:"text,omitempty"`
		Type      string   `json:"type"`
		MediaURLs []string `json:"media_urls"`
	}{
		From:      source,
		To:        destination,
		Text:      text,
		Type:      "MMS",
		MediaURLs: mediaURLs,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "error marshalling JSON")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telnyxMessagesURI, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "error creating request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.authtokenTelnyx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// the error of the http client names the request URL only, which is a constant.
		return nil, pkgerrors.Wrapf(err, "error making request")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ProviderError{Provider: providerTelnyx, StatusCode: resp.StatusCode}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "error reading response body")
	}

	var res telnyx.MessageResponse
	if errUnmarshal := json.Unmarshal(bodyBytes, &res); errUnmarshal != nil {
		return nil, pkgerrors.Wrapf(errUnmarshal, "error unmarshalling JSON")
	}

	return &res, nil
}

// MessagebirdSendMMS sends one MMS to the messagebird. The request has exactly one recipient
// because a messagebird request succeeds or fails as a whole for its recipient list.
func (h *requestExternal) MessagebirdSendMMS(ctx context.Context, sender string, destination string, text string, mediaURLs []string) (*messagebird.Message, error) {
	data := url.Values{}
	data.Set("recipients", destination)
	data.Set("originator", sender)
	if text != "" {
		data.Set("body", text)
	}
	for _, mediaURL := range mediaURLs {
		data.Add("mediaUrls[]", mediaURL)
	}
	encoded := data.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, messagebirdMMSURI, strings.NewReader(encoded))
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "could not create a request")
	}
	req.Header.Set("Authorization", "AccessKey "+h.authtokenMessagebird)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "could not send the request")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ProviderError{Provider: providerMessagebird, StatusCode: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "could not read the response")
	}

	var res messagebird.Message
	if errParse := json.Unmarshal(body, &res); errParse != nil {
		return nil, pkgerrors.Wrapf(errParse, "could not parse the response")
	}

	return &res, nil
}
