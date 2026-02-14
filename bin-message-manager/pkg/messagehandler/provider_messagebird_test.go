package messagehandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/messagebird"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

func TestNewMessageHandlerMessagebird(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)

	h := NewMessageHandlerMessagebird(mockExternal)

	if h == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestMessageHandlerMessagebird_SendMessage(t *testing.T) {
	messageID := uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name            string
		messageID       uuid.UUID
		source          *commonaddress.Address
		targets         []target.Target
		text            string
		messagebirdResp *messagebird.Message
		messagebirdErr  error
		expectError     bool
		expectCount     int
	}{
		{
			name:      "successful_send",
			messageID: messageID,
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+1234567890",
			},
			targets: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+0987654321",
					},
					Status: target.StatusQueued,
				},
			},
			text: "Test message",
			messagebirdResp: &messagebird.Message{
				ID: "msg-123",
				Recipients: messagebird.RecipientStruct{
					TotalCount: 1,
					Items: []messagebird.Recipient{
						{
							Recipient: 987654321,
							Status:    "sent",
						},
					},
				},
			},
			messagebirdErr: nil,
			expectError:    false,
			expectCount:    1,
		},
		{
			name:      "multiple_targets",
			messageID: messageID,
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+1234567890",
			},
			targets: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+0987654321",
					},
					Status: target.StatusQueued,
				},
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+1111111111",
					},
					Status: target.StatusQueued,
				},
			},
			text: "Test message",
			messagebirdResp: &messagebird.Message{
				ID: "msg-456",
				Recipients: messagebird.RecipientStruct{
					TotalCount: 2,
					Items: []messagebird.Recipient{
						{
							Recipient: 987654321,
							Status:    "sent",
						},
						{
							Recipient: 1111111111,
							Status:    "sent",
						},
					},
				},
			},
			messagebirdErr: nil,
			expectError:    false,
			expectCount:    2,
		},
		{
			name:      "messagebird_error",
			messageID: messageID,
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+1234567890",
			},
			targets: []target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+0987654321",
					},
					Status: target.StatusQueued,
				},
			},
			text:            "Test message",
			messagebirdResp: nil,
			messagebirdErr:  fmt.Errorf("messagebird api error"),
			expectError:     true,
			expectCount:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)

			h := &messageHandlerMessagebird{
				requestExternal: mockExternal,
			}

			ctx := context.Background()

			// Build expected receivers list
			receivers := []string{}
			for _, tgt := range tt.targets {
				receivers = append(receivers, tgt.Destination.Target)
			}

			mockExternal.EXPECT().
				MessagebirdSendMessage(ctx, tt.source.Target, receivers, tt.text).
				Return(tt.messagebirdResp, tt.messagebirdErr)

			result, err := h.SendMessage(ctx, tt.messageID, tt.source, tt.targets, tt.text)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != tt.expectCount {
				t.Errorf("Result count mismatch: got %d, want %d", len(result), tt.expectCount)
			}
		})
	}
}
