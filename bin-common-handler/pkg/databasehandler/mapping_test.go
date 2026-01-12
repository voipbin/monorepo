package databasehandler

import (
	"testing"

	"github.com/gofrs/uuid"
)

// Test model
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
			name:     "basic model with UUID and string fields",
			model:    &testModel{},
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
			name: "handles embedded structs",
			model: &struct {
				testModel
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
