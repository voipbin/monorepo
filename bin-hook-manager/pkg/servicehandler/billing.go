package servicehandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

// ValidationError indicates a client-side error (missing signature header, malformed body, a
// signature that billing rejected).
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

	// The signature is not verified here: the billing service is the only place that holds the
	// secret and verifies it. A request without the header can not be genuine, so it is rejected
	// before it is forwarded.
	sig := r.Header.Get("Paddle-Signature")
	if sig == "" {
		return &ValidationError{fmt.Errorf("missing Paddle-Signature header")}
	}

	req := &hmhook.Hook{
		ReceivedURI:  r.Host + r.URL.Path,
		ReceivedData: data,

		// The header is forwarded as received so that billing verifies it against the secret it
		// holds. The value is never logged by this function.
		ReceivedSignature: sig,
	}

	if err := h.reqHandler.BillingV1PaddleHook(ctx, req); err != nil {
		// Only an explicit rejection of the signature by billing (unauthorized) is a client
		// error. Every other failure, including a bad request answer, keeps the existing
		// behavior (not a ValidationError, so the HTTP layer answers 500).
		if errors.Is(err, requesthandler.ErrUnauthorized) {
			return &ValidationError{fmt.Errorf("billing rejected the webhook signature: %w", err)}
		}
		return fmt.Errorf("could not send the hook: %w", err)
	}

	return nil
}
