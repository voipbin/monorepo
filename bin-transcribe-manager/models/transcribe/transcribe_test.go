package transcribe

import (
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestTranscribeStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	onEndFlowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	hostID := uuid.Must(uuid.NewV4())
	streamingID1 := uuid.Must(uuid.NewV4())
	streamingID2 := uuid.Must(uuid.NewV4())

	tr := Transcribe{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ActiveflowID:  activeflowID,
		OnEndFlowID:   onEndFlowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusProgressing,
		HostID:        hostID,
		Language:      "en-US",
		Direction:     DirectionBoth,
		StreamingIDs:  []uuid.UUID{streamingID1, streamingID2},
		TMCreate:      "2023-01-01T00:00:00Z",
		TMUpdate:      "2023-01-02T00:00:00Z",
		TMDelete:      "",
	}

	if tr.ID != id {
		t.Errorf("Transcribe.ID = %v, expected %v", tr.ID, id)
	}
	if tr.CustomerID != customerID {
		t.Errorf("Transcribe.CustomerID = %v, expected %v", tr.CustomerID, customerID)
	}
	if tr.ActiveflowID != activeflowID {
		t.Errorf("Transcribe.ActiveflowID = %v, expected %v", tr.ActiveflowID, activeflowID)
	}
	if tr.OnEndFlowID != onEndFlowID {
		t.Errorf("Transcribe.OnEndFlowID = %v, expected %v", tr.OnEndFlowID, onEndFlowID)
	}
	if tr.ReferenceType != ReferenceTypeCall {
		t.Errorf("Transcribe.ReferenceType = %v, expected %v", tr.ReferenceType, ReferenceTypeCall)
	}
	if tr.ReferenceID != referenceID {
		t.Errorf("Transcribe.ReferenceID = %v, expected %v", tr.ReferenceID, referenceID)
	}
	if tr.Status != StatusProgressing {
		t.Errorf("Transcribe.Status = %v, expected %v", tr.Status, StatusProgressing)
	}
	if tr.HostID != hostID {
		t.Errorf("Transcribe.HostID = %v, expected %v", tr.HostID, hostID)
	}
	if tr.Language != "en-US" {
		t.Errorf("Transcribe.Language = %v, expected %v", tr.Language, "en-US")
	}
	if tr.Direction != DirectionBoth {
		t.Errorf("Transcribe.Direction = %v, expected %v", tr.Direction, DirectionBoth)
	}
	if len(tr.StreamingIDs) != 2 {
		t.Errorf("Transcribe.StreamingIDs length = %v, expected %v", len(tr.StreamingIDs), 2)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_unknown", ReferenceTypeUnknown, "unknown"},
		{"reference_type_recording", ReferenceTypeRecording, "recording"},
		{"reference_type_call", ReferenceTypeCall, "call"},
		{"reference_type_confbridge", ReferenceTypeConfbridge, "confbridge"},
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
		{"direction_both", DirectionBoth, "both"},
		{"direction_in", DirectionIn, "in"},
		{"direction_out", DirectionOut, "out"},
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
		{"status_done", StatusDone, "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestIsUpdatableStatus(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus Status
		newStatus Status
		expected  bool
	}{
		{"progressing_to_done", StatusProgressing, StatusDone, true},
		{"progressing_to_progressing", StatusProgressing, StatusProgressing, false},
		{"done_to_done", StatusDone, StatusDone, false},
		{"done_to_progressing", StatusDone, StatusProgressing, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUpdatableStatus(tt.oldStatus, tt.newStatus)
			if result != tt.expected {
				t.Errorf("IsUpdatableStatus(%s, %s) = %v, expected %v", tt.oldStatus, tt.newStatus, result, tt.expected)
			}
		})
	}
}
