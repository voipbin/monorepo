package confbridge

import (
	"testing"

	"github.com/gofrs/uuid"
)

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
