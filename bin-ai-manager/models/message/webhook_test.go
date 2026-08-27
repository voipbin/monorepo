package message

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"monorepo/bin-common-handler/models/eventtopic"
	"monorepo/bin-common-handler/models/identity"
)

// IntermediateWebhookMessage overrides the subscription address of the global topic exchange
// (VOIP-1404/1405). The assertion pins the POINTER type: the event data reaches notifyhandler as a
// POINTER and the assertion matches the dynamic type; a VALUE of this pointer-receiver type would
// fail the assertion (the exact pipecat defect this ticket fixed).
var _ eventtopic.SubscriptionIdentifier = (*IntermediateWebhookMessage)(nil)

func TestIntermediateWebhookMessageEventSubscriptionID(t *testing.T) {
	aicallID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name    string
		message *IntermediateWebhookMessage
		expect  string
	}{
		{
			"normal",
			&IntermediateWebhookMessage{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AIcallID:  aicallID,
				Role:      RoleAssistant,
				Content:   "hel",
				Direction: DirectionIncoming,
				Sequence:  1,
			},
			aicallID.String(),
		},
		{
			"empty aicall id",
			&IntermediateWebhookMessage{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.message.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

func TestIntermediateWebhookMessageEventSubscriptionIDIsNotOwnID(t *testing.T) {
	aicallID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	// Consecutive deltas of ONE utterance: each carries the per-delta id echoed from
	// bin-pipecat-manager and is ordered by Sequence. The id changes per event, so it is not an
	// address anybody could bind to -- the AIcall is.
	first := &IntermediateWebhookMessage{
		Identity: identity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: customerID,
		},
		AIcallID: aicallID,
		Role:     RoleAssistant,
		Content:  "hel",
		Sequence: 1,
	}
	second := &IntermediateWebhookMessage{
		Identity: identity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: customerID,
		},
		AIcallID: aicallID,
		Role:     RoleAssistant,
		Content:  "lo",
		Sequence: 2,
	}

	if first.ID == second.ID {
		t.Fatalf("Intermediate message ids are expected to differ per event. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != aicallID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", aicallID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the intermediate message own id. id: %s", first.ID)
	}
}

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
		{
			name: "converts_message_with_active_ai_id",
			message: &Message{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AIcallID:     uuid.Must(uuid.NewV4()),
				ActiveflowID: uuid.Must(uuid.NewV4()),
				ActiveAIID:   uuid.Must(uuid.NewV4()),
				Role:         RoleAssistant,
				Content:      "Test message with AI",
				Direction:    DirectionOutgoing,
				TMCreate:     ptrTime(time.Now()),
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, m *Message) {
				if wh.ActiveAIID != m.ActiveAIID {
					t.Errorf("Wrong ActiveAIID. expect: %s, got: %s", m.ActiveAIID, wh.ActiveAIID)
				}
				if wh.AIcallID != m.AIcallID {
					t.Errorf("Wrong AIcallID. expect: %s, got: %s", m.AIcallID, wh.AIcallID)
				}
				if wh.ActiveflowID != m.ActiveflowID {
					t.Errorf("Wrong ActiveflowID. expect: %s, got: %s", m.ActiveflowID, wh.ActiveflowID)
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
