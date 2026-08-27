package message

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	"monorepo/bin-common-handler/models/identity"
)

// Message overrides the subscription address of the global topic exchange (VOIP-1404/1405). The
// assertion pins the POINTER receiver: notifyhandler asserts on the dynamic type of the event
// data, which is always a pointer, so a value receiver would silently never be picked up.
var _ eventtopic.SubscriptionIdentifier = (*Message)(nil)

func TestMessageEventSubscriptionID(t *testing.T) {
	aicallID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name    string
		message *Message
		expect  string
	}{
		{
			"normal",
			&Message{
				Identity: identity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				AIcallID:  aicallID,
				Direction: DirectionIncoming,
				Role:      RoleAssistant,
				Content:   "hello",
			},
			aicallID.String(),
		},
		{
			"empty aicall id",
			&Message{},
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

func TestMessageEventSubscriptionIDIsNotOwnID(t *testing.T) {
	aicallID := uuid.Must(uuid.NewV4())

	first := &Message{
		Identity: identity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		AIcallID: aicallID,
		Role:     RoleUser,
		Content:  "first",
	}
	second := &Message{
		Identity: identity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: first.CustomerID,
		},
		AIcallID: aicallID,
		Role:     RoleAssistant,
		Content:  "second",
	}

	// Every message of one AIcall carries its own persisted id, which is exactly why the own id
	// must not be the subscription address: a subscriber following the conversation cannot know
	// those ids in advance. Both messages must resolve to the same address.
	if first.ID == second.ID {
		t.Fatalf("Message ids are expected to differ per message. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != aicallID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", aicallID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the message own id. id: %s", first.ID)
	}
}

func TestMessage_hasPipecatcallIDAndDeliveryStatus(t *testing.T) {
	m := Message{
		PipecatcallID:  uuid.Must(uuid.NewV4()),
		DeliveryStatus: DeliveryStatusPending,
	}
	if m.PipecatcallID == uuid.Nil {
		t.Fatal("PipecatcallID not set")
	}
	if m.DeliveryStatus != DeliveryStatusPending {
		t.Fatal("DeliveryStatus not set")
	}
}

func TestWebhookMessage_omitsInternalFields(t *testing.T) {
	m := Message{
		PipecatcallID:  uuid.Must(uuid.NewV4()),
		DeliveryStatus: DeliveryStatusDelivered,
	}
	wm := m.ConvertWebhookMessage()
	raw, _ := json.Marshal(wm)
	if strings.Contains(string(raw), "pipecatcall_id") {
		t.Fatalf("webhook leaks pipecatcall_id: %s", raw)
	}
	if strings.Contains(string(raw), "delivery_status") {
		t.Fatalf("webhook leaks delivery_status: %s", raw)
	}
}

func TestMessage(t *testing.T) {
	tests := []struct {
		name      string
		aicallID  uuid.UUID
		direction Direction
		role      Role
		content   string
	}{
		{
			name:      "creates_message_with_all_fields",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionIncoming,
			role:      RoleUser,
			content:   "Hello, how can I help you?",
		},
		{
			name:      "creates_message_with_empty_fields",
			aicallID:  uuid.Nil,
			direction: DirectionNone,
			role:      RoleNone,
			content:   "",
		},
		{
			name:      "creates_message_with_assistant_role",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionOutgoing,
			role:      RoleAssistant,
			content:   "I can help you with that.",
		},
		{
			name:      "creates_message_with_system_role",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionNone,
			role:      RoleSystem,
			content:   "System initialization complete",
		},
		{
			name:      "creates_message_with_tool_role",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionIncoming,
			role:      RoleTool,
			content:   "Tool execution result",
		},
		{
			name:      "creates_message_with_notification_role",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionOutgoing,
			role:      RoleNotification,
			content:   `{"type":"member_switched","transition_function_name":"transfer_to_sales"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			m := &Message{
				AIcallID:  tt.aicallID,
				Direction: tt.direction,
				Role:      tt.role,
				Content:   tt.content,
				TMCreate:  &now,
			}

			if m.AIcallID != tt.aicallID {
				t.Errorf("Wrong AIcallID. expect: %s, got: %s", tt.aicallID, m.AIcallID)
			}
			if m.Direction != tt.direction {
				t.Errorf("Wrong Direction. expect: %s, got: %s", tt.direction, m.Direction)
			}
			if m.Role != tt.role {
				t.Errorf("Wrong Role. expect: %s, got: %s", tt.role, m.Role)
			}
			if m.Content != tt.content {
				t.Errorf("Wrong Content. expect: %s, got: %s", tt.content, m.Content)
			}
		})
	}
}

func TestRoleConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Role
		expected string
	}{
		{
			name:     "role_none",
			constant: RoleNone,
			expected: "",
		},
		{
			name:     "role_system",
			constant: RoleSystem,
			expected: "system",
		},
		{
			name:     "role_user",
			constant: RoleUser,
			expected: "user",
		},
		{
			name:     "role_assistant",
			constant: RoleAssistant,
			expected: "assistant",
		},
		{
			name:     "role_function",
			constant: RoleFunction,
			expected: "function",
		},
		{
			name:     "role_tool",
			constant: RoleTool,
			expected: "tool",
		},
		{
			name:     "role_notification",
			constant: RoleNotification,
			expected: "notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDeliveryStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant DeliveryStatus
		expected string
	}{
		{
			name:     "delivery_status_pending",
			constant: DeliveryStatusPending,
			expected: "pending",
		},
		{
			name:     "delivery_status_delivered",
			constant: DeliveryStatusDelivered,
			expected: "delivered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDirectionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Direction
		expected string
	}{
		{
			name:     "direction_incoming",
			constant: DirectionIncoming,
			expected: "incoming",
		},
		{
			name:     "direction_outgoing",
			constant: DirectionOutgoing,
			expected: "outgoing",
		},
		{
			name:     "direction_none",
			constant: DirectionNone,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
