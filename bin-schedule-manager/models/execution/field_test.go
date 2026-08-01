package execution

import (
	"testing"
)

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{
			name:     "field_id",
			constant: FieldID,
			expected: "id",
		},
		{
			name:     "field_customer_id",
			constant: FieldCustomerID,
			expected: "customer_id",
		},
		{
			name:     "field_schedule_id",
			constant: FieldScheduleID,
			expected: "schedule_id",
		},
		{
			name:     "field_trigger_type",
			constant: FieldTriggerType,
			expected: "trigger_type",
		},
		{
			name:     "field_status",
			constant: FieldStatus,
			expected: "status",
		},
		{
			name:     "field_status_code",
			constant: FieldStatusCode,
			expected: "status_code",
		},
		{
			name:     "field_error",
			constant: FieldError,
			expected: "error",
		},
		{
			name:     "field_result",
			constant: FieldResult,
			expected: "result",
		},
		{
			name:     "field_attempt_count",
			constant: FieldAttemptCount,
			expected: "attempt_count",
		},
		{
			name:     "field_duration_ms",
			constant: FieldDurationMS,
			expected: "duration_ms",
		},
		{
			name:     "field_tm_scheduled",
			constant: FieldTMScheduled,
			expected: "tm_scheduled",
		},
		{
			name:     "field_tm_deadline",
			constant: FieldTMDeadline,
			expected: "tm_deadline",
		},
		{
			name:     "field_tm_start",
			constant: FieldTMStart,
			expected: "tm_start",
		},
		{
			name:     "field_tm_end",
			constant: FieldTMEnd,
			expected: "tm_end",
		},
		{
			name:     "field_tm_create",
			constant: FieldTMCreate,
			expected: "tm_create",
		},
		{
			name:     "field_tm_update",
			constant: FieldTMUpdate,
			expected: "tm_update",
		},
		{
			name:     "field_tm_delete",
			constant: FieldTMDelete,
			expected: "tm_delete",
		},
		{
			name:     "field_deleted",
			constant: FieldDeleted,
			expected: "deleted",
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
