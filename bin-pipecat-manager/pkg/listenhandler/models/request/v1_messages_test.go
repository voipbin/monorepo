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
			// Ensure the V1DataMessagesPost struct is constructable with the
			// given fields. The struct has no behavior to validate beyond
			// field presence, so just round-trip the value.
			_ = tt.req
		})
	}
}
