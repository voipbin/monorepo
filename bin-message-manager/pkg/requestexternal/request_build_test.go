package requestexternal

import (
	"context"
	"strings"
	"testing"
)

// Test_sendMessage_requestBuildFailure covers a request that cannot be created because the context is
// missing. No connection is attempted. The nil value is declared through a variable on purpose.
func Test_sendMessage_requestBuildFailure(t *testing.T) {
	var nilCtx context.Context

	h := NewRequestExternal("messagebird-token", "telnyx-token")

	t.Run("messagebird", func(t *testing.T) {
		res, err := h.MessagebirdSendMessage(nilCtx, "+15550000001", []string{"+15550000002"}, "hello")
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		if err == nil || !strings.HasPrefix(err.Error(), "Could not create a request.") {
			t.Errorf("Wrong match. expect: Could not create a request., got: %v", err)
		}
	})

	t.Run("telnyx", func(t *testing.T) {
		res, err := h.TelnyxSendMessage(nilCtx, "+15550000001", "+15550000002", "hello")
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		if err == nil || !strings.HasPrefix(err.Error(), "error creating request") {
			t.Errorf("Wrong match. expect: error creating request, got: %v", err)
		}
	})
}
