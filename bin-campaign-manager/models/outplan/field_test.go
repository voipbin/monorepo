package outplan

import (
	"testing"
)

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{"field_id", FieldID, "id"},
		{"field_customer_id", FieldCustomerID, "customer_id"},
		{"field_name", FieldName, "name"},
		{"field_detail", FieldDetail, "detail"},
		{"field_source", FieldSource, "source"},
		{"field_dial_timeout", FieldDialTimeout, "dial_timeout"},
		{"field_try_interval", FieldTryInterval, "try_interval"},
		{"field_max_try_count_0", FieldMaxTryCount0, "max_try_count_0"},
		{"field_max_try_count_1", FieldMaxTryCount1, "max_try_count_1"},
		{"field_max_try_count_2", FieldMaxTryCount2, "max_try_count_2"},
		{"field_max_try_count_3", FieldMaxTryCount3, "max_try_count_3"},
		{"field_max_try_count_4", FieldMaxTryCount4, "max_try_count_4"},
		{"field_tm_create", FieldTMCreate, "tm_create"},
		{"field_tm_update", FieldTMUpdate, "tm_update"},
		{"field_tm_delete", FieldTMDelete, "tm_delete"},
		{"field_deleted", FieldDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
