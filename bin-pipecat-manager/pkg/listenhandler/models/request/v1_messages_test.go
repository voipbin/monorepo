package request

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestV1DataMessagesPost_Struct(t *testing.T) {
	tests := []struct {
		name string
		req  V1DataMessagesPost
	}{
		{
			name: "full request",
			req: V1DataMessagesPost{
				PipecatcallID:  uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				MessageID:      "msg-123",
				MessageText:    "Hello, world!",
				RunImmediately: true,
				AudioResponse:  true,
			},
		},
		{
			name: "empty request",
			req:  V1DataMessagesPost{},
		},
		{
			name: "minimal request",
			req: V1DataMessagesPost{
				PipecatcallID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				MessageText:   "Test message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.PipecatcallID != tt.req.PipecatcallID {
				t.Errorf("PipecatcallID mismatch")
			}
			if tt.req.MessageID != tt.req.MessageID {
				t.Errorf("MessageID mismatch")
			}
			if tt.req.MessageText != tt.req.MessageText {
				t.Errorf("MessageText mismatch")
			}
			if tt.req.RunImmediately != tt.req.RunImmediately {
				t.Errorf("RunImmediately mismatch")
			}
			if tt.req.AudioResponse != tt.req.AudioResponse {
				t.Errorf("AudioResponse mismatch")
			}
		})
	}
}
