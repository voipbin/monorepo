package paddlehandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// signatureTolerance is the largest allowed distance between the timestamp of a webhook signature
// and the clock, in both directions.
const signatureTolerance = 300 * time.Second

var (
	// ErrSignatureInvalid is returned when the signature of a webhook is rejected: it does not match,
	// it is outside of the time window, it is malformed, or it is missing. The error holds no
	// signature, timestamp, body or key text.
	ErrSignatureInvalid = errors.New("the webhook signature is not valid")

	// ErrCredentialNotConfigured is returned when a Paddle credential is needed and no value is
	// configured for it: the webhook secret for the signature check, the API key for a portal session.
	// It is not ErrKeyUnavailable (a value that could not be read) and not a rejected signature.
	// The error holds no key text.
	ErrCredentialNotConfigured = errors.New("the paddle credential is not configured")

	// ErrKeyUnavailable is returned when a Paddle credential could not be read (a supplier that
	// can fail). The suppliers of the environment never return it. A credential that is simply
	// not configured is not an error. The error holds no key text.
	ErrKeyUnavailable = errors.New("the paddle key could not be read")
)

// readKey reads one credential at the time of use. A nil function means "not configured". An error
// is always reported as ErrKeyUnavailable, whatever the function returned.
func (h *paddleHandler) readKey(ctx context.Context, f KeyFunc) (string, error) {
	if f == nil {
		return "", nil
	}

	key, err := f(ctx)
	if err != nil {
		if errors.Is(err, ErrKeyUnavailable) {
			return "", err
		}
		return "", fmt.Errorf("%w: %w", ErrKeyUnavailable, err)
	}

	return key, nil
}

// VerifyWebhook checks the Paddle-Signature header of a webhook against the body that was
// received. The secret is read for every call and kept nowhere.
//
//   - No secret is configured: ErrCredentialNotConfigured. The webhook is rejected (fail closed),
//     whatever the signature is.
//   - The secret can not be read: ErrKeyUnavailable. The caller must not treat it as "no secret".
//   - A secret exists: nil for a valid signature, ErrSignatureInvalid for every other case.
func (h *paddleHandler) VerifyWebhook(ctx context.Context, signature string, body []byte) error {
	secret, err := h.readKey(ctx, h.webhookSecret)
	if err != nil {
		return err
	}

	if secret == "" {
		return fmt.Errorf("%w: webhook secret", ErrCredentialNotConfigured)
	}

	return verifySignature(secret, signature, body, time.Now())
}

// verifySignature verifies the Paddle-Signature header with HMAC-SHA256. The format is
// "ts=<timestamp>;h1=<hash>", and the hash is HMAC-SHA256(secret, ts + ":" + body) in lower case
// hex. The error carries a category only.
func verifySignature(secret, signature string, body []byte, now time.Time) error {
	if signature == "" {
		return fmt.Errorf("%w: missing", ErrSignatureInvalid)
	}

	parts := strings.Split(signature, ";")
	if len(parts) < 2 {
		return fmt.Errorf("%w: format", ErrSignatureInvalid)
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
		return fmt.Errorf("%w: format", ErrSignatureInvalid)
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: timestamp", ErrSignatureInvalid)
	}

	age := now.Unix() - tsInt
	limit := int64(signatureTolerance / time.Second)
	if age > limit || age < -limit {
		return fmt.Errorf("%w: timestamp", ErrSignatureInvalid)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":" + string(body)))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(h1)) {
		return fmt.Errorf("%w: mismatch", ErrSignatureInvalid)
	}

	return nil
}
