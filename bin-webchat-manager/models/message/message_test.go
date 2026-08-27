package message

import (
	"testing"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestMessageStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	sessionID := uuid.Must(uuid.NewV4())
	senderID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	m := Message{
		SessionID:    sessionID,
		Direction:    DirectionInbound,
		Status:       StatusSent,
		SenderID:     senderID,
		ActiveflowID: activeflowID,
		Text:         "hello",
	}
	m.ID = id
	m.CustomerID = customerID

	if m.ID != id {
		t.Errorf("Message.ID = %v, expected %v", m.ID, id)
	}
	if m.CustomerID != customerID {
		t.Errorf("Message.CustomerID = %v, expected %v", m.CustomerID, customerID)
	}
	if m.SessionID != sessionID {
		t.Errorf("Message.SessionID = %v, expected %v", m.SessionID, sessionID)
	}
	if m.Direction != DirectionInbound {
		t.Errorf("Message.Direction = %v, expected %v", m.Direction, DirectionInbound)
	}
	if m.Status != StatusSent {
		t.Errorf("Message.Status = %v, expected %v", m.Status, StatusSent)
	}
	if m.SenderID != senderID {
		t.Errorf("Message.SenderID = %v, expected %v", m.SenderID, senderID)
	}
	if m.ActiveflowID != activeflowID {
		t.Errorf("Message.ActiveflowID = %v, expected %v", m.ActiveflowID, activeflowID)
	}
	if m.Text != "hello" {
		t.Errorf("Message.Text = %v, expected %v", m.Text, "hello")
	}
}

func TestDirectionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Direction
		expected string
	}{
		{"direction_inbound", DirectionInbound, "inbound"},
		{"direction_outbound", DirectionOutbound, "outbound"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_sent", StatusSent, "sent"},
		{"status_delivered", StatusDelivered, "delivered"},
		{"status_failed", StatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

// Message overrides the subscription address of the global topic exchange (VOIP-1404/1405). The
// assertion pins the POINTER type: the event data reaches notifyhandler as a POINTER and the
// assertion matches the dynamic type; a VALUE of this pointer-receiver type would fail the
// assertion (the exact pipecat defect this ticket fixed).
var _ eventtopic.SubscriptionIdentifier = (*Message)(nil)

func TestMessageEventSubscriptionID(t *testing.T) {
	sessionID := uuid.FromStringOrNil("3c7f21a4-0000-4000-8000-000000000001")

	tests := []struct {
		name    string
		message *Message
		expect  string
	}{
		{
			name: "normal",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				WidgetID:  uuid.Must(uuid.NewV4()),
				SessionID: sessionID,
				Direction: DirectionInbound,
				Status:    StatusSent,
				Text:      "hello",
			},
			expect: sessionID.String(),
		},
		{
			name:    "empty session id",
			message: &Message{},
			expect:  uuid.Nil.String(),
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

// TestMessageEventSubscriptionIDIsNotOwnID is the load-bearing half of the pair: a Message DOES
// carry a stable own id, so an override that accidentally returned it would still look
// well-formed. Two distinct messages of ONE session must resolve to the same address, and that
// address must not be either message's own id. WidgetID is also set here on purpose — it is a
// denormalized convenience field and must never be mistaken for the address.
func TestMessageEventSubscriptionIDIsNotOwnID(t *testing.T) {
	sessionID := uuid.Must(uuid.NewV4())
	widgetID := uuid.Must(uuid.NewV4())

	first := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		WidgetID:  widgetID,
		SessionID: sessionID,
		Direction: DirectionInbound,
	}
	second := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: first.CustomerID,
		},
		WidgetID:  widgetID,
		SessionID: sessionID,
		Direction: DirectionOutbound,
	}

	if first.ID == second.ID {
		t.Fatalf("Message ids are expected to differ per message. id: %s", first.ID)
	}

	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the message's own id. id: %s", first.ID)
	}

	if first.EventSubscriptionID() == widgetID.String() {
		t.Errorf("Subscription address must not be the widget id. widget_id: %s", widgetID)
	}

	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
}
