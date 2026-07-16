package message

import (
	"testing"

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
