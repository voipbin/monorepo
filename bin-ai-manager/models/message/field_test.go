package message

import "testing"

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
			name:     "field_aicall_id",
			constant: FieldAIcallID,
			expected: "aicall_id",
		},
		{
			name:     "field_direction",
			constant: FieldDirection,
			expected: "direction",
		},
		{
			name:     "field_role",
			constant: FieldRole,
			expected: "role",
		},
		{
			name:     "field_content",
			constant: FieldContent,
			expected: "content",
		},
		{
			name:     "field_tool_calls",
			constant: FieldToolCalls,
			expected: "tool_calls",
		},
		{
			name:     "field_tool_call_id",
			constant: FieldToolCallID,
			expected: "tool_call_id",
		},
		{
			name:     "field_tm_create",
			constant: FieldTMCreate,
			expected: "tm_create",
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

// Test_FieldOrigin pins the origin field's database column name. It is compared
// against a literal because the value is written into SQL WHERE clauses by
// getPipecatcallMessages (via databasehandler.NotEq) -- a rename here without a
// matching migration is a silent query failure, not a compile error.
func Test_FieldOrigin(t *testing.T) {
	if FieldOrigin != "origin" {
		t.Errorf("FieldOrigin mismatch. expected: %q, got: %q", "origin", FieldOrigin)
	}
}

// Test_OriginValues pins the three Origin values. 'proactive' reaches the
// frontends (they badge on it) and 'listen_internal' reaches tenant webhook
// payloads, so both are external contract, not internal naming.
func Test_OriginValues(t *testing.T) {
	tests := []struct {
		name   string
		origin Origin
		expect string
	}{
		{"none", OriginNone, ""},
		{"proactive", OriginProactive, "proactive"},
		{"listen_internal", OriginListenInternal, "listen_internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.origin) != tt.expect {
				t.Errorf("Origin mismatch. expected: %q, got: %q", tt.expect, string(tt.origin))
			}
		})
	}
}
