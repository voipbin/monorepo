package service

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestServiceStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	s := Service{
		ID:          id,
		Type:        TypeAIcall,
		PushActions: nil,
	}

	if s.ID != id {
		t.Errorf("Service.ID = %v, expected %v", s.ID, id)
	}
	if s.Type != TypeAIcall {
		t.Errorf("Service.Type = %v, expected %v", s.Type, TypeAIcall)
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_aicall", TypeAIcall, "aicall"},
		{"type_ai_summary", TypeAISummary, "ai_summary"},
		{"type_conferencecall", TypeConferencecall, "conferencecall"},
		{"type_queuecall", TypeQueuecall, "queuecall"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestServiceWithDifferentTypes(t *testing.T) {
	tests := []struct {
		name        string
		serviceType Type
	}{
		{"aicall_service", TypeAIcall},
		{"ai_summary_service", TypeAISummary},
		{"conferencecall_service", TypeConferencecall},
		{"queuecall_service", TypeQueuecall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				ID:   uuid.Must(uuid.NewV4()),
				Type: tt.serviceType,
			}
			if s.Type != tt.serviceType {
				t.Errorf("Service.Type = %v, expected %v", s.Type, tt.serviceType)
			}
		})
	}
}
