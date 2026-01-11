package dbutil

import (
	"testing"

	"github.com/gofrs/uuid"
)

// Test model for validating dbutil functions
type testModel struct {
	ID     uuid.UUID `db:"id,uuid"`
	Name   string    `db:"name"`
	Count  int       `db:"count"`
	SkipMe bool      `db:"-"`
}

func TestGetDBFields_Basic(t *testing.T) {
	tests := []struct {
		name     string
		model    interface{}
		expected []string
	}{
		{
			name: "basic model with UUID and string fields",
			model: &testModel{},
			expected: []string{"id", "name", "count"},
		},
		{
			name: "skips fields with db:\"-\" tag",
			model: &struct {
				Field1 string `db:"field1"`
				Field2 string `db:"-"`
				Field3 string `db:"field3"`
			}{},
			expected: []string{"field1", "field3"},
		},
		{
			name: "handles fields without conversion type",
			model: &struct {
				ID   string `db:"id"`
				Name string `db:"name"`
			}{},
			expected: []string{"id", "name"},
		},
		{
			name: "handles fields with conversion types",
			model: &struct {
				ID   uuid.UUID `db:"id,uuid"`
				Data []string  `db:"data,json"`
			}{},
			expected: []string{"id", "data"},
		},
		{
			name: "handles embedded structs recursively",
			model: &struct {
				testModel  // embedded struct
				Extra string `db:"extra"`
			}{},
			expected: []string{"id", "name", "count", "extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDBFields(tt.model)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d", len(tt.expected), len(result))
				return
			}

			for i, field := range result {
				if field != tt.expected[i] {
					t.Errorf("field[%d]: expected %s, got %s", i, tt.expected[i], field)
				}
			}
		})
	}
}

func TestPrepareValues_Basic(t *testing.T) {
	tests := []struct {
		name     string
		model    interface{}
		expected []interface{}
	}{
		{
			name: "basic string and int fields",
			model: &struct {
				Name  string `db:"name"`
				Count int    `db:"count"`
			}{
				Name:  "test",
				Count: 42,
			},
			expected: []interface{}{"test", 42},
		},
		{
			name: "skips fields with db:\"-\" tag",
			model: &struct {
				Field1 string `db:"field1"`
				Field2 string `db:"-"`
				Field3 string `db:"field3"`
			}{
				Field1: "value1",
				Field2: "skip",
				Field3: "value3",
			},
			expected: []interface{}{"value1", "value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareValues(tt.model)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d values, got %d", len(tt.expected), len(result))
				return
			}

			for i, val := range result {
				if val != tt.expected[i] {
					t.Errorf("value[%d]: expected %v, got %v", i, tt.expected[i], val)
				}
			}
		})
	}
}

func TestScanRow_Basic(t *testing.T) {
	t.Skip("Not implemented yet")
}
