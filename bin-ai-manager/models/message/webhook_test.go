package message

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"monorepo/bin-common-handler/models/identity"
)

func TestConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name      string
		message   *Message
		wantNil   bool
		checkFunc func(t *testing.T, wh *WebhookMessage, m *Message)
	}{
		{
			name: "converts_message_with_all_fields",
			message: &Message{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AIcallID:   uuid.Must(uuid.NewV4()),
				Role:       RoleUser,
				Content:    "Test message",
				Direction:  DirectionIncoming,
				ToolCalls:  []ToolCall{{ID: "call_123", Type: ToolTypeFunction}},
				ToolCallID: "call_456",
				TMCreate:   ptrTime(time.Now()),
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, m *Message) {
				if wh.ID != m.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", m.ID, wh.ID)
				}
				if wh.CustomerID != m.CustomerID {
					t.Errorf("Wrong CustomerID. expect: %s, got: %s", m.CustomerID, wh.CustomerID)
				}
				if wh.AIcallID != m.AIcallID {
					t.Errorf("Wrong AIcallID. expect: %s, got: %s", m.AIcallID, wh.AIcallID)
				}
				if wh.Role != m.Role {
					t.Errorf("Wrong Role. expect: %s, got: %s", m.Role, wh.Role)
				}
				if wh.Content != m.Content {
					t.Errorf("Wrong Content. expect: %s, got: %s", m.Content, wh.Content)
				}
				if wh.Direction != m.Direction {
					t.Errorf("Wrong Direction. expect: %s, got: %s", m.Direction, wh.Direction)
				}
				if wh.ToolCallID != m.ToolCallID {
					t.Errorf("Wrong ToolCallID. expect: %s, got: %s", m.ToolCallID, wh.ToolCallID)
				}
			},
		},
		{
			name: "converts_message_with_empty_fields",
			message: &Message{
				Identity: identity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
				AIcallID:  uuid.Nil,
				Role:      RoleNone,
				Content:   "",
				Direction: DirectionNone,
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, m *Message) {
				if wh.ID != m.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", m.ID, wh.ID)
				}
				if wh.Role != m.Role {
					t.Errorf("Wrong Role. expect: %s, got: %s", m.Role, wh.Role)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wh := tt.message.ConvertWebhookMessage()
			if wh == nil && !tt.wantNil {
				t.Error("Expected non-nil webhook message, got nil")
				return
			}
			if wh != nil && tt.wantNil {
				t.Error("Expected nil webhook message, got non-nil")
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, wh, tt.message)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name      string
		message   *Message
		wantError bool
	}{
		{
			name: "creates_webhook_event_successfully",
			message: &Message{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AIcallID:  uuid.Must(uuid.NewV4()),
				Role:      RoleAssistant,
				Content:   "Response message",
				Direction: DirectionOutgoing,
				TMCreate:  ptrTime(time.Now()),
			},
			wantError: false,
		},
		{
			name: "creates_webhook_event_with_empty_message",
			message: &Message{
				Identity: identity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.message.CreateWebhookEvent()
			if (err != nil) != tt.wantError {
				t.Errorf("CreateWebhookEvent() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				// Verify it's valid JSON
				var wh WebhookMessage
				if errUnmarshal := json.Unmarshal(data, &wh); errUnmarshal != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", errUnmarshal)
				}
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
