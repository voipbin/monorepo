package confbridge

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Both event wrappers embed Confbridge BY VALUE, so they satisfy the interface via method
// promotion through the embedded Confbridge's commonidentity.Identity (VOIP-1419); no wrapper
// writes its own method. The assertions below are the insurance: a future same-depth embed that
// silently dropped or ambiguated the promoted method would break them. The assertions pin the
// POINTER types, matching how the event data reaches notifyhandler.
var (
	_ eventtopic.SubscriptionIdentifier = (*EventConfbridgeJoined)(nil)
	_ eventtopic.SubscriptionIdentifier = (*EventConfbridgeLeaved)(nil)
)

// TestEventConfbridgeWrappersEventSubscriptionID mutation-checks the one WRONG implementation the
// call-manager golden suite guards against: each wrapper carries a second uuid (the joined/leaved
// call id) that is a plausible-looking but wrong address. The subscription address must be the
// embedded confbridge's own id.
func TestEventConfbridgeWrappersEventSubscriptionID(t *testing.T) {
	confbridgeID := uuid.Must(uuid.NewV4())
	callID := uuid.Must(uuid.NewV4())

	base := Confbridge{
		Identity: commonidentity.Identity{
			ID:         confbridgeID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
	}

	tests := []struct {
		name string
		data eventtopic.SubscriptionIdentifier
	}{
		{"joined", &EventConfbridgeJoined{Confbridge: base, JoinedCallID: callID}},
		{"leaved", &EventConfbridgeLeaved{Confbridge: base, LeavedCallID: callID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.data.EventSubscriptionID()
			if res != confbridgeID.String() {
				t.Errorf("Wrong match. expect: %s, got: %s", confbridgeID, res)
			}
			if res == callID.String() {
				t.Errorf("Subscription address must not be the joined/leaved call id. call_id: %s", callID)
			}
		})
	}
}

// TestEventConfbridgeWrappersEventSubscriptionIDEmpty pins the zero-value degrade path: an unset
// embedded confbridge resolves to the uuid.Nil string, which the routing-key layer collapses to
// the `-` placeholder.
func TestEventConfbridgeWrappersEventSubscriptionIDEmpty(t *testing.T) {
	tests := []struct {
		name string
		data eventtopic.SubscriptionIdentifier
	}{
		{"joined", &EventConfbridgeJoined{}},
		{"leaved", &EventConfbridgeLeaved{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.data.EventSubscriptionID()
			if res != uuid.Nil.String() {
				t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil, res)
			}
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_confbridge_created", EventTypeConfbridgeCreated, "confbridge_created"},
		{"event_type_confbridge_deleted", EventTypeConfbridgeDeleted, "confbridge_deleted"},
		{"event_type_confbridge_terminating", EventTypeConfbridgeTerminating, "confbridge_terminating"},
		{"event_type_confbridge_terminated", EventTypeConfbridgeTerminated, "confbridge_terminated"},
		{"event_type_confbridge_joined", EventTypeConfbridgeJoined, "confbridge_joined"},
		{"event_type_confbridge_leaved", EventTypeConfbridgeLeaved, "confbridge_leaved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestEventConfbridgeLeavedStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	leavedCallID := uuid.Must(uuid.NewV4())

	e := EventConfbridgeLeaved{
		LeavedCallID: leavedCallID,
	}
	e.ID = id

	if e.ID != id {
		t.Errorf("EventConfbridgeLeaved.ID = %v, expected %v", e.ID, id)
	}
	if e.LeavedCallID != leavedCallID {
		t.Errorf("EventConfbridgeLeaved.LeavedCallID = %v, expected %v", e.LeavedCallID, leavedCallID)
	}
}

func TestEventConfbridgeJoinedStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	joinedCallID := uuid.Must(uuid.NewV4())

	e := EventConfbridgeJoined{
		JoinedCallID: joinedCallID,
	}
	e.ID = id

	if e.ID != id {
		t.Errorf("EventConfbridgeJoined.ID = %v, expected %v", e.ID, id)
	}
	if e.JoinedCallID != joinedCallID {
		t.Errorf("EventConfbridgeJoined.JoinedCallID = %v, expected %v", e.JoinedCallID, joinedCallID)
	}
}
