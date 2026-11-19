package servicehandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"monorepo/bin-common-handler/pkg/requesthandler"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

const (
	// messageTargetTwilio is the end of the path of the twilio delivery callback.
	messageTargetTwilio = "/twilio"

	// twilioSignatureHeader is the header that carries the signature of a twilio callback.
	twilioSignatureHeader = "X-Twilio-Signature"

	// maxTwilioBodySize is the largest body of a twilio callback that is read (a status callback is a short form).
	maxTwilioBodySize = 64 * 1024
)

// Message handles message receive
func (h *serviceHandler) Message(ctx context.Context, r *http.Request) error {
	if strings.HasSuffix(r.URL.Path, messageTargetTwilio) {
		return h.messageTwilio(ctx, r)
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("could not read request body: %w", err)
	}

	req := &hmhook.Hook{
		ReceivedURI:  r.Host + r.URL.Path,
		ReceivedData: data,
	}

	if err := h.reqHandler.MessageV1Hook(ctx, req); err != nil {
		return fmt.Errorf("could not send the hook: %w", err)
	}

	return nil
}

// messageTwilio handles a twilio delivery callback.
//
// The signature is not verified here: the message-manager is the only place that holds the Auth Token and checks it
// against the callback URL that it gave to Twilio. A request without the header can not be genuine, so it is refused
// before it is forwarded. The header is forwarded as received and is never logged by this function.
//
// A refused callback is a ValidationError, which the HTTP layer answers as a client error (400) and not as a server
// fault (500): the sender of the request is refused, nothing failed inside.
func (h *serviceHandler) messageTwilio(ctx context.Context, r *http.Request) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxTwilioBodySize+1))
	if err != nil {
		return &ValidationError{fmt.Errorf("could not read request body: %w", err)}
	}
	if len(data) > maxTwilioBodySize {
		return &ValidationError{fmt.Errorf("request body exceeds maximum size of %d bytes", maxTwilioBodySize)}
	}

	signature := r.Header.Get(twilioSignatureHeader)
	if signature == "" {
		return &ValidationError{fmt.Errorf("missing %s header", twilioSignatureHeader)}
	}

	req := &hmhook.Hook{
		ReceivedURI:       r.Host + r.URL.Path,
		ReceivedData:      data,
		ReceivedSignature: signature,
	}

	if err := h.reqHandler.MessageV1Hook(ctx, req); err != nil {
		if errors.Is(err, requesthandler.ErrUnauthorized) {
			return &ValidationError{fmt.Errorf("message-manager refused the callback: %w", err)}
		}
		return fmt.Errorf("could not send the hook: %w", err)
	}

	return nil
}
