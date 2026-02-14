package messagehandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/models/telnyx"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

func TestNewMessageHandlerTelnyx(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)

	h := NewMessageHandlerTelnyx(mockExternal)

	if h == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestMessageHandlerTelnyx_SendMessage(t *testing.T) {
	messageID := uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name         string
		messageID    uuid.UUID
		source       *commonaddress.Address
		targets      []target.Target
		text         string
		telnyxResp   *telnyx.MessageResponse
		telnyxErr    error
		expectError  bool
		expectCount  int
	}{
		{
			name:      "successful_single_send",
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
			telnyxResp: &telnyx.MessageResponse{
				Data: telnyx.MessageData{
					ID: "telnyx-msg-123",
				},
			},
			telnyxErr:   nil,
			expectError: false,
			expectCount: 1,
		},
		{
			name:      "telnyx_error",
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
			text:        "Test message",
			telnyxResp:  nil,
			telnyxErr:   fmt.Errorf("telnyx api error"),
			expectError: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)

			h := &messageHandlerTelnyx{
				requestExternal: mockExternal,
			}

			ctx := context.Background()

			// Setup expectations for each target
			for _, tgt := range tt.targets {
				mockExternal.EXPECT().
					TelnyxSendMessage(ctx, tt.source.Target, tgt.Destination.Target, tt.text).
					Return(tt.telnyxResp, tt.telnyxErr)
			}

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

func TestMessageHandlerTelnyx_SendMessage_MultipleTargets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)

	h := &messageHandlerTelnyx{
		requestExternal: mockExternal,
	}

	ctx := context.Background()
	messageID := uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000")

	source := &commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+1234567890",
	}

	targets := []target.Target{
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
	}

	text := "Test message to multiple recipients"

	// Mock successful responses for both targets
	for _, tgt := range targets {
		mockExternal.EXPECT().
			TelnyxSendMessage(ctx, source.Target, tgt.Destination.Target, text).
			Return(&telnyx.MessageResponse{
				Data: telnyx.MessageData{
					ID: "telnyx-msg-" + tgt.Destination.Target,
				},
			}, nil)
	}

	result, err := h.SendMessage(ctx, messageID, source, targets, text)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
}
