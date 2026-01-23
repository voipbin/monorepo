package streaming

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestStreaming(t *testing.T) {
	tests := []struct {
		name string

		podID         string
		activeflowID  uuid.UUID
		referenceType ReferenceType
		referenceID   uuid.UUID
		language      string
		gender        Gender
		direction     Direction
		messageID     uuid.UUID
		vendorName    VendorName
	}{
		{
			name: "creates_streaming_with_all_fields",

			podID:         "pod-12345",
			activeflowID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			referenceType: ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			language:      "en-US",
			gender:        GenderMale,
			direction:     DirectionOutgoing,
			messageID:     uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
			vendorName:    VendorNameElevenlabs,
		},
		{
			name: "creates_streaming_with_empty_fields",

			podID:         "",
			activeflowID:  uuid.Nil,
			referenceType: ReferenceTypeNone,
			referenceID:   uuid.Nil,
			language:      "",
			gender:        "",
			direction:     DirectionNone,
			messageID:     uuid.Nil,
			vendorName:    VendorNameNone,
		},
		{
			name: "creates_streaming_with_confbridge_reference",

			podID:         "pod-67890",
			activeflowID:  uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
			referenceType: ReferenceTypeConfbridge,
			referenceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
			language:      "ko-KR",
			gender:        GenderFemale,
			direction:     DirectionBoth,
			messageID:     uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440006"),
			vendorName:    VendorNameElevenlabs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Streaming{
				PodID:         tt.podID,
				ActiveflowID:  tt.activeflowID,
				ReferenceType: tt.referenceType,
				ReferenceID:   tt.referenceID,
				Language:      tt.language,
				Gender:        tt.gender,
				Direction:     tt.direction,
				MessageID:     tt.messageID,
				VendorName:    tt.vendorName,
			}

			if s.PodID != tt.podID {
				t.Errorf("Wrong PodID. expect: %s, got: %s", tt.podID, s.PodID)
			}
			if s.ActiveflowID != tt.activeflowID {
				t.Errorf("Wrong ActiveflowID. expect: %s, got: %s", tt.activeflowID, s.ActiveflowID)
			}
			if s.ReferenceType != tt.referenceType {
				t.Errorf("Wrong ReferenceType. expect: %s, got: %s", tt.referenceType, s.ReferenceType)
			}
			if s.ReferenceID != tt.referenceID {
				t.Errorf("Wrong ReferenceID. expect: %s, got: %s", tt.referenceID, s.ReferenceID)
			}
			if s.Language != tt.language {
				t.Errorf("Wrong Language. expect: %s, got: %s", tt.language, s.Language)
			}
			if s.Gender != tt.gender {
				t.Errorf("Wrong Gender. expect: %s, got: %s", tt.gender, s.Gender)
			}
			if s.Direction != tt.direction {
				t.Errorf("Wrong Direction. expect: %s, got: %s", tt.direction, s.Direction)
			}
			if s.MessageID != tt.messageID {
				t.Errorf("Wrong MessageID. expect: %s, got: %s", tt.messageID, s.MessageID)
			}
			if s.VendorName != tt.vendorName {
				t.Errorf("Wrong VendorName. expect: %s, got: %s", tt.vendorName, s.VendorName)
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
		{
			name:     "direction_none",
			constant: DirectionNone,
			expected: "",
		},
		{
			name:     "direction_incoming",
			constant: DirectionIncoming,
			expected: "in",
		},
		{
			name:     "direction_outgoing",
			constant: DirectionOutgoing,
			expected: "out",
		},
		{
			name:     "direction_both",
			constant: DirectionBoth,
			expected: "both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestGenderConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Gender
		expected string
	}{
		{
			name:     "gender_male",
			constant: GenderMale,
			expected: "male",
		},
		{
			name:     "gender_female",
			constant: GenderFemale,
			expected: "female",
		},
		{
			name:     "gender_neutral",
			constant: GenderNeutral,
			expected: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{
			name:     "reference_type_none",
			constant: ReferenceTypeNone,
			expected: "",
		},
		{
			name:     "reference_type_call",
			constant: ReferenceTypeCall,
			expected: "call",
		},
		{
			name:     "reference_type_confbridge",
			constant: ReferenceTypeConfbridge,
			expected: "confbridge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestVendorNameConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant VendorName
		expected string
	}{
		{
			name:     "vendor_name_none",
			constant: VendorNameNone,
			expected: "",
		},
		{
			name:     "vendor_name_elevenlabs",
			constant: VendorNameElevenlabs,
			expected: "elevenlabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
