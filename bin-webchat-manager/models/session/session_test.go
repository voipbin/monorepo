package session

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestSessionStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	widgetID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	s := Session{
		WidgetID:     widgetID,
		Status:       StatusActive,
		ActiveflowID: activeflowID,
	}
	s.ID = id
	s.CustomerID = customerID

	if s.ID != id {
		t.Errorf("Session.ID = %v, expected %v", s.ID, id)
	}
	if s.CustomerID != customerID {
		t.Errorf("Session.CustomerID = %v, expected %v", s.CustomerID, customerID)
	}
	if s.WidgetID != widgetID {
		t.Errorf("Session.WidgetID = %v, expected %v", s.WidgetID, widgetID)
	}
	if s.Status != StatusActive {
		t.Errorf("Session.Status = %v, expected %v", s.Status, StatusActive)
	}
	if s.ActiveflowID != activeflowID {
		t.Errorf("Session.ActiveflowID = %v, expected %v", s.ActiveflowID, activeflowID)
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_active", StatusActive, "active"},
		{"status_ended", StatusEnded, "ended"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
