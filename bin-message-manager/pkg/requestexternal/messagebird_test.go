package requestexternal

import (
	"context"
	"errors"
	"testing"
)

func Test_MessagebirdSendMessage_canceledContext(t *testing.T) {
	h := NewRequestExternal("token", "token")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// the request carries the caller's context, so a canceled context fails before any network use
	_, err := h.MessagebirdSendMessage(ctx, "+15550100", []string{"+15550101"}, "hello")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("MessagebirdSendMessage() error = %v, want it to wrap context.Canceled", err)
	}
}
