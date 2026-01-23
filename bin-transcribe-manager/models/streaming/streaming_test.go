package streaming

import (
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"
)

func TestStreamingStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	transcribeID := uuid.Must(uuid.NewV4())

	s := Streaming{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	if s.ID != id {
		t.Errorf("Streaming.ID = %v, expected %v", s.ID, id)
	}
	if s.CustomerID != customerID {
		t.Errorf("Streaming.CustomerID = %v, expected %v", s.CustomerID, customerID)
	}
	if s.TranscribeID != transcribeID {
		t.Errorf("Streaming.TranscribeID = %v, expected %v", s.TranscribeID, transcribeID)
	}
	if s.Language != "en-US" {
		t.Errorf("Streaming.Language = %v, expected %v", s.Language, "en-US")
	}
	if s.Direction != transcript.DirectionIn {
		t.Errorf("Streaming.Direction = %v, expected %v", s.Direction, transcript.DirectionIn)
	}
}

func TestStreamingWithDifferentDirections(t *testing.T) {
	tests := []struct {
		name      string
		direction transcript.Direction
	}{
		{"direction_in", transcript.DirectionIn},
		{"direction_out", transcript.DirectionOut},
		{"direction_both", transcript.DirectionBoth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Streaming{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				TranscribeID: uuid.Must(uuid.NewV4()),
				Language:     "ko-KR",
				Direction:    tt.direction,
			}

			if s.Direction != tt.direction {
				t.Errorf("Streaming.Direction = %v, expected %v", s.Direction, tt.direction)
			}
		})
	}
}
