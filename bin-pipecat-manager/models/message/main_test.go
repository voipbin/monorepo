package message

import (
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

func TestMessage_Struct(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "full message",
			msg: Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},
				PipecatcallID:            uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
				PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
				Text:                     "Hello, world!",
			},
		},
		{
			name: "empty message",
			msg:  Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.msg.ID != tt.msg.ID {
				t.Errorf("ID mismatch")
			}
			if tt.msg.Text != tt.msg.Text {
				t.Errorf("Text mismatch")
			}
		})
	}
}
