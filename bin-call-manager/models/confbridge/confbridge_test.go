package confbridge

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestConfbridgeStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	recordingID := uuid.Must(uuid.NewV4())
	externalMediaID := uuid.Must(uuid.NewV4())

	c := Confbridge{
		ActiveflowID:    activeflowID,
		ReferenceType:   ReferenceTypeConference,
		ReferenceID:     referenceID,
		Type:            TypeConference,
		Status:          StatusProgressing,
		BridgeID:        "bridge-123",
		Flags:           []Flag{FlagNoAutoLeave},
		ChannelCallIDs:  map[string]uuid.UUID{"channel-1": uuid.Must(uuid.NewV4())},
		RecordingID:     recordingID,
		RecordingIDs:    []uuid.UUID{recordingID},
		ExternalMediaID: externalMediaID,
	}
	c.ID = id

	if c.ID != id {
		t.Errorf("Confbridge.ID = %v, expected %v", c.ID, id)
	}
	if c.ActiveflowID != activeflowID {
		t.Errorf("Confbridge.ActiveflowID = %v, expected %v", c.ActiveflowID, activeflowID)
	}
	if c.ReferenceType != ReferenceTypeConference {
		t.Errorf("Confbridge.ReferenceType = %v, expected %v", c.ReferenceType, ReferenceTypeConference)
	}
	if c.ReferenceID != referenceID {
		t.Errorf("Confbridge.ReferenceID = %v, expected %v", c.ReferenceID, referenceID)
	}
	if c.Type != TypeConference {
		t.Errorf("Confbridge.Type = %v, expected %v", c.Type, TypeConference)
	}
	if c.Status != StatusProgressing {
		t.Errorf("Confbridge.Status = %v, expected %v", c.Status, StatusProgressing)
	}
	if c.BridgeID != "bridge-123" {
		t.Errorf("Confbridge.BridgeID = %v, expected %v", c.BridgeID, "bridge-123")
	}
	if len(c.Flags) != 1 {
		t.Errorf("Confbridge.Flags length = %v, expected %v", len(c.Flags), 1)
	}
	if len(c.ChannelCallIDs) != 1 {
		t.Errorf("Confbridge.ChannelCallIDs length = %v, expected %v", len(c.ChannelCallIDs), 1)
	}
	if c.RecordingID != recordingID {
		t.Errorf("Confbridge.RecordingID = %v, expected %v", c.RecordingID, recordingID)
	}
	if len(c.RecordingIDs) != 1 {
		t.Errorf("Confbridge.RecordingIDs length = %v, expected %v", len(c.RecordingIDs), 1)
	}
	if c.ExternalMediaID != externalMediaID {
		t.Errorf("Confbridge.ExternalMediaID = %v, expected %v", c.ExternalMediaID, externalMediaID)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_call", ReferenceTypeCall, "call"},
		{"reference_type_conference", ReferenceTypeConference, "conference"},
		{"reference_type_ai", ReferenceTypeAI, "ai"},
		{"reference_type_queue", ReferenceTypeQueue, "queue"},
		{"reference_transcribe", ReferenceTranscribe, "transcribe"},
		{"reference_transfer", ReferenceTransfer, "transfer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_connect", TypeConnect, "connect"},
		{"type_conference", TypeConference, "conference"},
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
		{"status_progressing", StatusProgressing, "progressing"},
		{"status_terminating", StatusTerminating, "terminating"},
		{"status_terminated", StatusTerminated, "terminated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestFlagConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Flag
		expected string
	}{
		{"flag_no_auto_leave", FlagNoAutoLeave, "no_auto_leave"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
