package stream

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestStreamStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	s := Stream{
		ID:            id,
		Encapsulation: EncapsulationAudiosocket,
	}

	if s.ID != id {
		t.Errorf("Stream.ID = %v, expected %v", s.ID, id)
	}
	if s.Encapsulation != EncapsulationAudiosocket {
		t.Errorf("Stream.Encapsulation = %v, expected %v", s.Encapsulation, EncapsulationAudiosocket)
	}
}

func TestEncapsulationConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Encapsulation
		expected string
	}{
		{"encapsulation_none", EncapsulationNone, ""},
		{"encapsulation_audiosocket", EncapsulationAudiosocket, "audiosocket"},
		{"encapsulation_rtp", EncapsulationRTP, "rtp"},
		{"encapsulation_sln", EncapsulationSLN, "sln"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
