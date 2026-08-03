package webhookhandler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// signatureHeader is the HTTP header carrying the HMAC-SHA256 signature of
// the outbound webhook body, so receivers can verify the request originated
// from VoIPBin and was not tampered with in transit.
const signatureHeader = "X-VoIPBIN-Signature"

// computeSignature returns the hex-encoded HMAC-SHA256 of body keyed by secret,
// formatted as "sha256=<hex>" (mirrors the convention already used elsewhere
// in the monorepo for verifying inbound signatures, e.g. Meta's X-Hub-Signature-256).
func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// sendMessage sends the message to the given uri with the given method and data.
// When secret is non-empty, the request is signed and delivered with an
// X-VoIPBIN-Signature header so the receiver can verify authenticity.
func (h *webhookHandler) sendMessage(uri string, method string, dataType string, data []byte, secret string) error {

	log := logrus.WithFields(
		logrus.Fields{
			"func":   "sendMessage",
			"uri":    uri,
			"method": method,
		},
	)
	log.Debugf("Sending a message.")

	// Validate the webhook URL before attempting delivery.
	if err := validateWebhookURL(uri); err != nil {
		log.Errorf("Webhook URL validation failed. err: %v", err)
		return fmt.Errorf("webhook URL validation failed: %w", err)
	}

	// Guard against nil httpClient (e.g. in unit tests that construct the struct directly).
	// Always use the safe client to maintain SSRF protection.
	client := h.httpClient
	if client == nil {
		client = newSafeHTTPClient()
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(method, uri, bytes.NewBuffer(data))
		if err != nil {
			log.Errorf("Could not create request. err: %v", err)
			return err
		}

		if data != nil && dataType != "" {
			req.Header.Set("Content-Type", dataType)
		}

		if secret != "" {
			req.Header.Set(signatureHeader, computeSignature(secret, data))
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Could not send the request correctly. Making a retrying: %d, err: %v", i, err)
			lastErr = err
			time.Sleep(time.Second * 1)
			continue
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 500 {
			log.Errorf("Received server error. Making a retrying: %d, status: %d", i, resp.StatusCode)
			lastErr = fmt.Errorf("server returned status %d", resp.StatusCode)
			time.Sleep(time.Second * 1)
			continue
		}

		log.WithField("response_status", resp.StatusCode).Debugf("Sent the event correctly.")
		return nil
	}

	log.Errorf("Could not send the request. err: %v", lastErr)
	return lastErr
}
