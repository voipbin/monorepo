package conferencecall

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestConferencecallStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	conferenceID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	cc := Conferencecall{
		ActiveflowID:  activeflowID,
		ConferenceID:  conferenceID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusJoining,
		TMCreate:      "2024-01-01 00:00:00.000000",
		TMUpdate:      "2024-01-01 00:00:00.000000",
		TMDelete:      "9999-01-01 00:00:00.000000",
	}
	cc.ID = id

	if cc.ID != id {
		t.Errorf("Conferencecall.ID = %v, expected %v", cc.ID, id)
	}
	if cc.ActiveflowID != activeflowID {
		t.Errorf("Conferencecall.ActiveflowID = %v, expected %v", cc.ActiveflowID, activeflowID)
	}
	if cc.ConferenceID != conferenceID {
		t.Errorf("Conferencecall.ConferenceID = %v, expected %v", cc.ConferenceID, conferenceID)
	}
	if cc.ReferenceType != ReferenceTypeCall {
		t.Errorf("Conferencecall.ReferenceType = %v, expected %v", cc.ReferenceType, ReferenceTypeCall)
	}
	if cc.Status != StatusJoining {
		t.Errorf("Conferencecall.Status = %v, expected %v", cc.Status, StatusJoining)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_call", ReferenceTypeCall, "call"},
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
		{"status_joining", StatusJoining, "joining"},
		{"status_joined", StatusJoined, "joined"},
		{"status_leaving", StatusLeaving, "leaving"},
		{"status_leaved", StatusLeaved, "leaved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
