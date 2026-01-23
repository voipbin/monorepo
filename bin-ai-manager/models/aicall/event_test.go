package aicall

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "event_type_status_initializing",
			constant: EventTypeStatusInitializing,
			expected: "aicall_status_initializing",
		},
		{
			name:     "event_type_status_progressing",
			constant: EventTypeStatusProgressing,
			expected: "aicall_status_progressing",
		},
		{
			name:     "event_type_status_pausing",
			constant: EventTypeStatusPausing,
			expected: "aicall_status_pausing",
		},
		{
			name:     "event_type_status_resuming",
			constant: EventTypeStatusResuming,
			expected: "aicall_status_resuming",
		},
		{
			name:     "event_type_status_terminating",
			constant: EventTypeStatusTerminating,
			expected: "aicall_status_terminating",
		},
		{
			name:     "event_type_status_terminated",
			constant: EventTypeStatusTerminated,
			expected: "aicall_status_terminated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
