package summary

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestSummary(t *testing.T) {
	tests := []struct {
		name          string
		activeflowID  uuid.UUID
		onEndFlowID   uuid.UUID
		referenceType ReferenceType
		referenceID   uuid.UUID
		status        Status
		language      string
		content       string
	}{
		{
			name:          "creates_summary_with_all_fields",
			activeflowID:  uuid.Must(uuid.NewV4()),
			onEndFlowID:   uuid.Must(uuid.NewV4()),
			referenceType: ReferenceTypeCall,
			referenceID:   uuid.Must(uuid.NewV4()),
			status:        StatusProgressing,
			language:      "en",
			content:       "This is a test summary",
		},
		{
			name:          "creates_summary_with_empty_fields",
			activeflowID:  uuid.Nil,
			onEndFlowID:   uuid.Nil,
			referenceType: ReferenceTypeNone,
			referenceID:   uuid.Nil,
			status:        StatusNone,
			language:      "",
			content:       "",
		},
		{
			name:          "creates_summary_with_conference_reference",
			activeflowID:  uuid.Must(uuid.NewV4()),
			onEndFlowID:   uuid.Must(uuid.NewV4()),
			referenceType: ReferenceTypeConference,
			referenceID:   uuid.Must(uuid.NewV4()),
			status:        StatusDone,
			language:      "es",
			content:       "Conference summary complete",
		},
		{
			name:          "creates_summary_with_transcribe_reference",
			activeflowID:  uuid.Must(uuid.NewV4()),
			onEndFlowID:   uuid.Must(uuid.NewV4()),
			referenceType: ReferenceTypeTranscribe,
			referenceID:   uuid.Must(uuid.NewV4()),
			status:        StatusProgressing,
			language:      "fr",
			content:       "Transcription in progress",
		},
		{
			name:          "creates_summary_with_recording_reference",
			activeflowID:  uuid.Must(uuid.NewV4()),
			onEndFlowID:   uuid.Must(uuid.NewV4()),
			referenceType: ReferenceTypeRecording,
			referenceID:   uuid.Must(uuid.NewV4()),
			status:        StatusDone,
			language:      "de",
			content:       "Recording summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			s := &Summary{
				ActiveflowID:  tt.activeflowID,
				OnEndFlowID:   tt.onEndFlowID,
				ReferenceType: tt.referenceType,
				ReferenceID:   tt.referenceID,
				Status:        tt.status,
				Language:      tt.language,
				Content:       tt.content,
				TMCreate:      &now,
			}

			if s.ActiveflowID != tt.activeflowID {
				t.Errorf("Wrong ActiveflowID. expect: %s, got: %s", tt.activeflowID, s.ActiveflowID)
			}
			if s.OnEndFlowID != tt.onEndFlowID {
				t.Errorf("Wrong OnEndFlowID. expect: %s, got: %s", tt.onEndFlowID, s.OnEndFlowID)
			}
			if s.ReferenceType != tt.referenceType {
				t.Errorf("Wrong ReferenceType. expect: %s, got: %s", tt.referenceType, s.ReferenceType)
			}
			if s.ReferenceID != tt.referenceID {
				t.Errorf("Wrong ReferenceID. expect: %s, got: %s", tt.referenceID, s.ReferenceID)
			}
			if s.Status != tt.status {
				t.Errorf("Wrong Status. expect: %s, got: %s", tt.status, s.Status)
			}
			if s.Language != tt.language {
				t.Errorf("Wrong Language. expect: %s, got: %s", tt.language, s.Language)
			}
			if s.Content != tt.content {
				t.Errorf("Wrong Content. expect: %s, got: %s", tt.content, s.Content)
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
			name:     "reference_type_conference",
			constant: ReferenceTypeConference,
			expected: "conference",
		},
		{
			name:     "reference_type_transcribe",
			constant: ReferenceTypeTranscribe,
			expected: "transcribe",
		},
		{
			name:     "reference_type_recording",
			constant: ReferenceTypeRecording,
			expected: "recording",
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

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{
			name:     "status_none",
			constant: StatusNone,
			expected: "",
		},
		{
			name:     "status_progressing",
			constant: StatusProgressing,
			expected: "progressing",
		},
		{
			name:     "status_done",
			constant: StatusDone,
			expected: "done",
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
