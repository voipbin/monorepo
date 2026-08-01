package schedule

import "testing"

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		expected string
	}{
		{
			name:     "event_type_schedule_created",
			event:    EventTypeScheduleCreated,
			expected: "schedule_created",
		},
		{
			name:     "event_type_schedule_updated",
			event:    EventTypeScheduleUpdated,
			expected: "schedule_updated",
		},
		{
			name:     "event_type_schedule_deleted",
			event:    EventTypeScheduleDeleted,
			expected: "schedule_deleted",
		},
		{
			name:     "event_type_execution_succeeded",
			event:    EventTypeExecutionSucceeded,
			expected: "execution_succeeded",
		},
		{
			name:     "event_type_execution_failed",
			event:    EventTypeExecutionFailed,
			expected: "execution_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event != tt.expected {
				t.Errorf("Wrong event type. expect: %s, got: %s", tt.expected, tt.event)
			}
		})
	}
}
