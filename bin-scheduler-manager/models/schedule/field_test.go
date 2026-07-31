package schedule

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
			name:     "field_name",
			constant: FieldName,
			expected: "name",
		},
		{
			name:     "field_detail",
			constant: FieldDetail,
			expected: "detail",
		},
		{
			name:     "field_type",
			constant: FieldType,
			expected: "type",
		},
		{
			name:     "field_cron",
			constant: FieldCron,
			expected: "cron",
		},
		{
			name:     "field_target_queue",
			constant: FieldTargetQueue,
			expected: "target_queue",
		},
		{
			name:     "field_target_uri",
			constant: FieldTargetURI,
			expected: "target_uri",
		},
		{
			name:     "field_target_method",
			constant: FieldTargetMethod,
			expected: "target_method",
		},
		{
			name:     "field_target_data_type",
			constant: FieldTargetDataType,
			expected: "target_data_type",
		},
		{
			name:     "field_target_data",
			constant: FieldTargetData,
			expected: "target_data",
		},
		{
			name:     "field_timeout_ms",
			constant: FieldTimeoutMS,
			expected: "timeout_ms",
		},
		{
			name:     "field_retry_max",
			constant: FieldRetryMax,
			expected: "retry_max",
		},
		{
			name:     "field_enabled",
			constant: FieldEnabled,
			expected: "enabled",
		},
		{
			name:     "field_tm_next_run",
			constant: FieldTMNextRun,
			expected: "tm_next_run",
		},
		{
			name:     "field_tm_last_run",
			constant: FieldTMLastRun,
			expected: "tm_last_run",
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
