package session

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Compile-time proof that Session participates in the explicit subscription-address
// contract of the global topic exchange (VOIP-1419): publishing it requires
// eventtopic.SubscriptionIdentifier, and this assertion survives even if every
// publish site is later removed.
var _ eventtopic.SubscriptionIdentifier = (*Session)(nil)

// TestSessionEventSubscriptionIDIsOwnID pins the address to the Session's OWN id,
// mutation-checked against the plausible wrong answers: every id-shaped field carries a
// distinct UUID, so returning CustomerID, WidgetID, or ActiveflowID instead of ID fails.
// Session.ID doubles as the visitor's continuity token, which is exactly why it is the
// subscription address a subscriber holds before any messages flow.
func TestSessionEventSubscriptionIDIsOwnID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	widgetID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	s := &Session{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		WidgetID:     widgetID,
		Status:       StatusEnded,
		ActiveflowID: activeflowID,
	}

	res := s.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Subscription address must not be the customer id. got: %s", res)
	}
	if res == widgetID.String() {
		t.Errorf("Subscription address must not be the widget id. got: %s", res)
	}
	if res == activeflowID.String() {
		t.Errorf("Subscription address must not be the activeflow id. got: %s", res)
	}
}

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
