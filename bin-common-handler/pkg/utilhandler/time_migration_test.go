package utilhandler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Test that *time.Time marshals to ISO 8601 format (RFC 3339)
func Test_TimePointer_JSONMarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    *time.Time
		expected string
	}{
		{
			name:     "non-nil time marshals to ISO 8601",
			input:    func() *time.Time { t := time.Date(2026, 2, 5, 10, 30, 45, 123456000, time.UTC); return &t }(),
			expected: `"2026-02-05T10:30:45.123456Z"`,
		},
		{
			name:     "nil time marshals to null",
			input:    nil,
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("got %s, want %s", string(result), tt.expected)
			}
		})
	}
}

// Test that *time.Time unmarshals from ISO 8601 format
func Test_TimePointer_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectNil bool
		expectErr bool
	}{
		{
			name:      "ISO 8601 with Z",
			input:     `"2026-02-05T10:30:45.123456Z"`,
			expectNil: false,
			expectErr: false,
		},
		{
			name:      "null becomes nil",
			input:     `null`,
			expectNil: true,
			expectErr: false,
		},
		{
			name:      "ISO 8601 without microseconds",
			input:     `"2026-02-05T10:30:45Z"`,
			expectNil: false,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *time.Time
			err := json.Unmarshal([]byte(tt.input), &result)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectNil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
			if !tt.expectNil && result == nil {
				t.Error("expected non-nil, got nil")
			}
		})
	}
}

// Test struct with timestamp fields marshals correctly
func Test_StructWithTimestamps_JSONMarshal(t *testing.T) {
	type TestModel struct {
		ID       string     `json:"id"`
		TMCreate *time.Time `json:"tm_create,omitempty"`
		TMUpdate *time.Time `json:"tm_update,omitempty"`
		TMDelete *time.Time `json:"tm_delete,omitempty"`
	}

	now := time.Date(2026, 2, 5, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name     string
		input    TestModel
		contains []string
		excludes []string
	}{
		{
			name: "all timestamps set",
			input: TestModel{
				ID:       "test-1",
				TMCreate: &now,
				TMUpdate: &now,
				TMDelete: &now,
			},
			contains: []string{`"tm_create"`, `"tm_update"`, `"tm_delete"`},
			excludes: []string{},
		},
		{
			name: "only tm_create set (omitempty hides nil)",
			input: TestModel{
				ID:       "test-2",
				TMCreate: &now,
				TMUpdate: nil,
				TMDelete: nil,
			},
			contains: []string{`"tm_create"`},
			excludes: []string{`"tm_update"`, `"tm_delete"`},
		},
		{
			name: "all nil (omitempty hides all)",
			input: TestModel{
				ID:       "test-3",
				TMCreate: nil,
				TMUpdate: nil,
				TMDelete: nil,
			},
			contains: []string{`"id"`},
			excludes: []string{`"tm_create"`, `"tm_update"`, `"tm_delete"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			resultStr := string(result)
			for _, s := range tt.contains {
				if !strings.Contains(resultStr, s) {
					t.Errorf("expected to contain %s, got %s", s, resultStr)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(resultStr, s) {
					t.Errorf("expected to NOT contain %s, got %s", s, resultStr)
				}
			}
		})
	}
}

// Test time comparison helpers
func Test_TimeComparison(t *testing.T) {
	now := time.Now().UTC()
	earlier := now.Add(-time.Hour)
	later := now.Add(time.Hour)

	tests := []struct {
		name    string
		a       *time.Time
		b       *time.Time
		aIsNil  bool
		bIsNil  bool
		aAfterB bool
	}{
		{"both nil", nil, nil, true, true, false},
		{"a nil, b not", nil, &now, true, false, false},
		{"a not, b nil", &now, nil, false, true, false},
		{"a before b", &earlier, &later, false, false, false},
		{"a after b", &later, &earlier, false, false, true},
		{"a equals b", &now, &now, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.a == nil) != tt.aIsNil {
				t.Errorf("a nil check failed")
			}
			if (tt.b == nil) != tt.bIsNil {
				t.Errorf("b nil check failed")
			}
			if tt.a != nil && tt.b != nil {
				if tt.a.After(*tt.b) != tt.aAfterB {
					t.Errorf("After comparison failed")
				}
			}
		})
	}
}

// Test struct with timestamp fields unmarshals correctly
func Test_StructWithTimestamps_JSONUnmarshal(t *testing.T) {
	type TestModel struct {
		ID       string     `json:"id"`
		TMCreate *time.Time `json:"tm_create,omitempty"`
		TMUpdate *time.Time `json:"tm_update,omitempty"`
		TMDelete *time.Time `json:"tm_delete,omitempty"`
	}

	tests := []struct {
		name             string
		input            string
		expectTMCreate   bool
		expectTMUpdate   bool
		expectTMDelete   bool
	}{
		{
			name:             "all timestamps present",
			input:            `{"id":"test-1","tm_create":"2026-02-05T10:30:45Z","tm_update":"2026-02-05T10:30:45Z","tm_delete":"2026-02-05T10:30:45Z"}`,
			expectTMCreate:   true,
			expectTMUpdate:   true,
			expectTMDelete:   true,
		},
		{
			name:             "only tm_create present",
			input:            `{"id":"test-2","tm_create":"2026-02-05T10:30:45Z"}`,
			expectTMCreate:   true,
			expectTMUpdate:   false,
			expectTMDelete:   false,
		},
		{
			name:             "timestamps explicitly null",
			input:            `{"id":"test-3","tm_create":null,"tm_update":null,"tm_delete":null}`,
			expectTMCreate:   false,
			expectTMUpdate:   false,
			expectTMDelete:   false,
		},
		{
			name:             "no timestamps in JSON",
			input:            `{"id":"test-4"}`,
			expectTMCreate:   false,
			expectTMUpdate:   false,
			expectTMDelete:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestModel
			err := json.Unmarshal([]byte(tt.input), &result)
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if (result.TMCreate != nil) != tt.expectTMCreate {
				t.Errorf("TMCreate: got nil=%v, want nil=%v", result.TMCreate == nil, !tt.expectTMCreate)
			}
			if (result.TMUpdate != nil) != tt.expectTMUpdate {
				t.Errorf("TMUpdate: got nil=%v, want nil=%v", result.TMUpdate == nil, !tt.expectTMUpdate)
			}
			if (result.TMDelete != nil) != tt.expectTMDelete {
				t.Errorf("TMDelete: got nil=%v, want nil=%v", result.TMDelete == nil, !tt.expectTMDelete)
			}
		})
	}
}

// Test that zero time is different from nil
func Test_ZeroTimeVsNil(t *testing.T) {
	var nilTime *time.Time
	zeroTimePtr := createZeroTimePtr()

	// nil time
	if nilTime != nil {
		t.Error("nilTime should be nil")
	}

	// zero time pointer is not nil - verified by isNotNil helper
	if !isNotNil(zeroTimePtr) {
		t.Error("zeroTimePtr should not be nil")
	}

	// zero time IsZero() returns true
	if !zeroTimePtr.IsZero() {
		t.Error("zeroTimePtr.IsZero() should be true")
	}

	// Marshaling behavior
	nilJSON, _ := json.Marshal(nilTime)
	zeroJSON, _ := json.Marshal(zeroTimePtr)

	if string(nilJSON) != "null" {
		t.Errorf("nil time should marshal to null, got %s", string(nilJSON))
	}

	// Zero time marshals to a specific timestamp, not null
	if string(zeroJSON) == "null" {
		t.Error("zero time should NOT marshal to null")
	}
}

// createZeroTimePtr returns a pointer to zero time.
// This helper exists to avoid staticcheck SA4031 warning.
func createZeroTimePtr() *time.Time {
	zeroTime := time.Time{}
	return &zeroTime
}

// isNotNil checks if a time pointer is not nil.
// This helper exists to avoid staticcheck SA4031 warning.
func isNotNil(t *time.Time) bool {
	return t != nil
}
