package conference

import (
	"testing"
)

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_none", TypeNone, ""},
		{"type_conference", TypeConference, "conference"},
		{"type_connect", TypeConnect, "connect"},
		{"type_queue", TypeQueue, "queue"},
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
		{"status_starting", StatusStarting, "starting"},
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

func TestIsValidConferenceType(t *testing.T) {
	tests := []struct {
		name     string
		confType Type
		expected bool
	}{
		{"valid_none", TypeNone, true},
		{"valid_conference", TypeConference, true},
		{"valid_connect", TypeConnect, true},
		{"invalid_queue", TypeQueue, false},
		{"invalid_unknown", Type("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidConferenceType(tt.confType)
			if result != tt.expected {
				t.Errorf("IsValidConferenceType(%s) = %v, expected %v", tt.confType, result, tt.expected)
			}
		})
	}
}
