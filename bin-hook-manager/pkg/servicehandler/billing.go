package servicehandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	hmhook "monorepo/bin-hook-manager/models/hook"

	"github.com/sirupsen/logrus"
)

// paddleSignatureMaxAge is the maximum allowed age (in seconds) for a Paddle webhook signature.
// Signatures older than this are rejected to prevent replay attacks.
const paddleSignatureMaxAge = 300 // 5 minutes

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

	// Reject stale or future-dated signatures to prevent replay attacks
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp in Paddle-Signature: %w", err)
	}
	age := time.Now().Unix() - tsInt
	if age > paddleSignatureMaxAge || age < -paddleSignatureMaxAge {
		return fmt.Errorf("Paddle-Signature timestamp out of valid range (age: %ds, max: %ds)", age, paddleSignatureMaxAge)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":" + string(body)))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedMAC), []byte(h1)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ValidationError indicates a client-side error (bad signature, malformed body).
// The HTTP handler uses this to return 400 instead of 500.
type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

// maxWebhookBodySize is the maximum allowed webhook body size (1 MB).
const maxWebhookBodySize = 1 << 20

// Billing handles billing webhook receive
func (h *serviceHandler) Billing(ctx context.Context, r *http.Request) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize+1))
	if err != nil {
		return &ValidationError{fmt.Errorf("could not read request body: %w", err)}
	}
	if int64(len(data)) > maxWebhookBodySize {
		return &ValidationError{fmt.Errorf("request body exceeds maximum size of %d bytes", maxWebhookBodySize)}
	}

	// Verify Paddle webhook signature — fail-closed when secret is not configured.
	// Without a valid secret, any HTTP client could submit fake billing events.
	if h.paddleWebhookSecret == "" {
		return &ValidationError{fmt.Errorf("paddle webhook secret not configured; rejecting request")}
	}
	sig := r.Header.Get("Paddle-Signature")
	if sig == "" {
		return &ValidationError{fmt.Errorf("missing Paddle-Signature header")}
	}
	if err := verifyPaddleSignature(h.paddleWebhookSecret, sig, data); err != nil {
		log := logrus.WithFields(logrus.Fields{
			"func": "Billing",
		})
		log.Errorf("Paddle webhook signature verification failed. signature: %s, body: %s, err: %v", sig, string(data), err)
		return &ValidationError{fmt.Errorf("webhook signature verification failed: %w", err)}
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
