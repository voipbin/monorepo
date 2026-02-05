package transcript

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestTranscriptStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	transcribeID := uuid.Must(uuid.NewV4())

	tmTranscript := time.Date(2023, 1, 1, 0, 0, 1, 123456000, time.UTC)
	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tr := Transcript{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Direction:    DirectionIn,
		Message:      "Hello, this is a test message.",
		TMTranscript: &tmTranscript,
		TMCreate:     &tmCreate,
		TMDelete:     nil,
	}

	if tr.ID != id {
		t.Errorf("Transcript.ID = %v, expected %v", tr.ID, id)
	}
	if tr.CustomerID != customerID {
		t.Errorf("Transcript.CustomerID = %v, expected %v", tr.CustomerID, customerID)
	}
	if tr.TranscribeID != transcribeID {
		t.Errorf("Transcript.TranscribeID = %v, expected %v", tr.TranscribeID, transcribeID)
	}
	if tr.Direction != DirectionIn {
		t.Errorf("Transcript.Direction = %v, expected %v", tr.Direction, DirectionIn)
	}
	if tr.Message != "Hello, this is a test message." {
		t.Errorf("Transcript.Message = %v, expected %v", tr.Message, "Hello, this is a test message.")
	}
	if tr.TMTranscript == nil || !tr.TMTranscript.Equal(tmTranscript) {
		t.Errorf("Transcript.TMTranscript = %v, expected %v", tr.TMTranscript, tmTranscript)
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
