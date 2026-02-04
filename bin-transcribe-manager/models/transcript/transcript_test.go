package transcript

import (
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestTranscriptStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	transcribeID := uuid.Must(uuid.NewV4())

	tr := Transcript{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Direction:    DirectionIn,
		Message:      "Hello, this is a test message.",
		TMTranscript: "2023-01-01T00:00:01.123456Z",
		TMCreate:     "2023-01-01T00:00:00Z",
		TMDelete:     "",
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
	if tr.TMTranscript != "2023-01-01T00:00:01.123456Z" {
		t.Errorf("Transcript.TMTranscript = %v, expected %v", tr.TMTranscript, "2023-01-01T00:00:01.123456Z")
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
