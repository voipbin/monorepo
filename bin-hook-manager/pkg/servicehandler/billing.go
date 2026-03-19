package servicehandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

// verifyPaddleSignature verifies the Paddle-Signature header using HMAC-SHA256.
// Paddle v2 format: "ts=<timestamp>;h1=<hash>"
// Verification: HMAC-SHA256(secret, ts + ":" + rawBody)
func verifyPaddleSignature(secret string, signature string, body []byte) error {
	parts := strings.Split(signature, ";")
	if len(parts) < 2 {
		return fmt.Errorf("invalid Paddle-Signature format")
	}

	var ts, h1 string
	for _, part := range parts {
		if strings.HasPrefix(part, "ts=") {
			ts = strings.TrimPrefix(part, "ts=")
		} else if strings.HasPrefix(part, "h1=") {
			h1 = strings.TrimPrefix(part, "h1=")
		}
	}

	if ts == "" || h1 == "" {
		return fmt.Errorf("missing ts or h1 in Paddle-Signature")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":" + string(body)))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedMAC), []byte(h1)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// Billing handles billing webhook receive
func (h *serviceHandler) Billing(ctx context.Context, r *http.Request) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("could not read request body: %w", err)
	}

	// Verify Paddle webhook signature if secret is configured
	if h.paddleWebhookSecret != "" {
		sig := r.Header.Get("Paddle-Signature")
		if sig == "" {
			return fmt.Errorf("missing Paddle-Signature header")
		}
		if err := verifyPaddleSignature(h.paddleWebhookSecret, sig, data); err != nil {
			return fmt.Errorf("webhook signature verification failed: %w", err)
		}
	}

	req := &hmhook.Hook{
		ReceviedURI:  r.Host + r.URL.Path,
		ReceivedData: data,
	}

	if err := h.reqHandler.BillingV1PaddleHook(ctx, req); err != nil {
		return fmt.Errorf("could not send the hook: %w", err)
	}

	return nil
}
