package databasehandler

import (
	"reflect"
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

func TestGetDBFields_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		model    interface{}
		expected []string
	}{
		{
			name:     "non-pointer struct input",
			model:    testModel{},
			expected: []string{"id", "name", "count"},
		},
		{
			name:     "nil pointer input",
			model:    (*testModel)(nil),
			expected: []string{},
		},
		{
			name: "empty struct with no db tags",
			model: &struct {
				Field1 string
				Field2 int
				Field3 bool
			}{},
			expected: []string{},
		},
		{
			name: "struct with only unexported fields",
			model: &struct {
				privateField string `db:"private_field"`
				anotherOne   int    `db:"another_one"`
			}{},
			expected: []string{},
		},
		{
			name: "struct with mixed exported and unexported fields",
			model: &struct {
				PublicField  string `db:"public_field"`
				privateField string `db:"private_field"`
				AnotherOne   int    `db:"another_one"`
			}{},
			expected: []string{"public_field", "another_one"},
		},
		{
			name: "multiple levels of embedded structs",
			model: &struct {
				Level1 struct {
					Level2 struct {
						DeepField string `db:"deep_field"`
					}
					MidField string `db:"mid_field"`
				}
				TopField string `db:"top_field"`
			}{},
			expected: []string{"top_field"},
		},
		{
			name: "deeply nested anonymous embedded structs",
			model: &struct {
				testModel
				Nested struct {
					testModel
					Inner string `db:"inner"`
				}
				Final string `db:"final"`
			}{},
			expected: []string{"id", "name", "count", "final"},
		},
		{
			name: "different conversion types - uuid",
			model: &struct {
				ID1 uuid.UUID `db:"id1,uuid"`
				ID2 uuid.UUID `db:"id2,uuid"`
			}{},
			expected: []string{"id1", "id2"},
		},
		{
			name: "different conversion types - json",
			model: &struct {
				Data1 string `db:"data1,json"`
				Data2 string `db:"data2,json"`
			}{},
			expected: []string{"data1", "data2"},
		},
		{
			name: "mixed conversion types",
			model: &struct {
				ID       uuid.UUID `db:"id,uuid"`
				Data     string    `db:"data,json"`
				Name     string    `db:"name"`
				Settings string    `db:"settings,json"`
			}{},
			expected: []string{"id", "data", "name", "settings"},
		},
		{
			name: "fields without db tags are skipped",
			model: &struct {
				Field1 string `db:"field1"`
				Field2 string
				Field3 string `db:"field3"`
				Field4 int
			}{},
			expected: []string{"field1", "field3"},
		},
		{
			name: "empty struct pointer",
			model: &struct {
			}{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDBFields(tt.model)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d. Expected: %v, Got: %v",
					len(tt.expected), len(result), tt.expected, result)
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

func TestGetDBFields_NilPointerPanic(t *testing.T) {
	// Test that nil pointer doesn't cause panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetDBFields panicked with nil pointer: %v", r)
		}
	}()

	result := GetDBFields((*testModel)(nil))
	if len(result) != 0 {
		t.Errorf("expected empty result for nil pointer, got %v", result)
	}
}

func TestConvertValue(t *testing.T) {
	tests := []struct {
		name           string
		value          interface{}
		conversionType string
		wantType       string // Expected type name
		wantErr        bool
	}{
		{
			name:           "UUID to bytes",
			value:          uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000")),
			conversionType: "uuid",
			wantType:       "[]uint8",
			wantErr:        false,
		},
		{
			name:           "slice to JSON",
			value:          []string{"a", "b", "c"},
			conversionType: "json",
			wantType:       "[]uint8",
			wantErr:        false,
		},
		{
			name:           "primitive passthrough",
			value:          "test string",
			conversionType: "",
			wantType:       "string",
			wantErr:        false,
		},
		{
			name:           "int passthrough",
			value:          42,
			conversionType: "",
			wantType:       "int",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertValue(tt.value, tt.conversionType)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				gotType := reflect.TypeOf(result).String()
				if gotType != tt.wantType {
					t.Errorf("convertValue() type = %v, want %v", gotType, tt.wantType)
				}
			}
		})
	}
}
