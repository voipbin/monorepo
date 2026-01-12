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

func TestPrepareFieldsFromStruct(t *testing.T) {
	id := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	tests := []struct {
		name      string
		input     interface{}
		wantKeys  []string
		wantErr   bool
		checkFunc func(t *testing.T, result map[string]any)
	}{
		{
			name: "basic struct with primitives",
			input: &struct {
				Name  string `db:"name"`
				Count int    `db:"count"`
			}{
				Name:  "test",
				Count: 42,
			},
			wantKeys: []string{"name", "count"},
			checkFunc: func(t *testing.T, result map[string]any) {
				if result["name"] != "test" {
					t.Errorf("name = %v, want test", result["name"])
				}
				if result["count"] != 42 {
					t.Errorf("count = %v, want 42", result["count"])
				}
			},
		},
		{
			name: "struct with UUID conversion",
			input: &struct {
				ID uuid.UUID `db:"id,uuid"`
			}{
				ID: id,
			},
			wantKeys: []string{"id"},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["id"].([]byte)
				if !ok {
					t.Errorf("id type = %T, want []byte", result["id"])
				}
				if len(bytes) != 16 {
					t.Errorf("id length = %d, want 16", len(bytes))
				}
			},
		},
		{
			name: "struct with JSON conversion",
			input: &struct {
				Tags []string `db:"tags,json"`
			}{
				Tags: []string{"a", "b"},
			},
			wantKeys: []string{"tags"},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["tags"].([]byte)
				if !ok {
					t.Errorf("tags type = %T, want []byte", result["tags"])
				}
				if string(bytes) != `["a","b"]` {
					t.Errorf("tags = %s, want [\"a\",\"b\"]", string(bytes))
				}
			},
		},
		{
			name: "skips db:\"-\" fields",
			input: &struct {
				Name   string `db:"name"`
				Secret string `db:"-"`
			}{
				Name:   "test",
				Secret: "hidden",
			},
			wantKeys: []string{"name"},
			checkFunc: func(t *testing.T, result map[string]any) {
				if _, exists := result["Secret"]; exists {
					t.Errorf("Secret field should be skipped")
				}
			},
		},
		// CRITICAL: Embedded struct handling test
		{
			name: "handles embedded structs",
			input: &struct {
				testModel // Embedded struct
				Extra     string `db:"extra"`
			}{
				testModel: testModel{
					ID:    id,
					Name:  "embedded",
					Count: 99,
				},
				Extra: "more",
			},
			wantKeys: []string{"id", "name", "count", "extra"},
			checkFunc: func(t *testing.T, result map[string]any) {
				if result["name"] != "embedded" {
					t.Errorf("name = %v, want embedded", result["name"])
				}
				if result["count"] != 99 {
					t.Errorf("count = %v, want 99", result["count"])
				}
				if result["extra"] != "more" {
					t.Errorf("extra = %v, want more", result["extra"])
				}
				// Verify UUID conversion
				bytes, ok := result["id"].([]byte)
				if !ok {
					t.Errorf("id type = %T, want []byte", result["id"])
				}
				if len(bytes) != 16 {
					t.Errorf("id length = %d, want 16", len(bytes))
				}
			},
		},
		// CRITICAL: Error propagation from embedded struct conversion
		{
			name: "error in embedded struct conversion",
			input: func() interface{} {
				// Create embedded struct with invalid UUID conversion
				type embeddedBadUUID struct {
					ID string `db:"id,uuid"` // string can't convert to UUID
				}
				result := &struct {
					embeddedBadUUID
				}{
					embeddedBadUUID: embeddedBadUUID{
						ID: "not-a-uuid",
					},
				}
				return result
			}(),
			wantErr: true,
		},
		// Nice to have: Unexported fields test
		{
			name: "skips unexported fields",
			input: &struct {
				Name   string `db:"name"`
				secret string `db:"secret"` // unexported
			}{
				Name:   "test",
				secret: "hidden",
			},
			wantKeys: []string{"name"},
			checkFunc: func(t *testing.T, result map[string]any) {
				if _, exists := result["secret"]; exists {
					t.Errorf("secret field should be skipped")
				}
			},
		},
		// Nice to have: Fields without db tags test
		{
			name: "skips fields without db tags",
			input: &struct {
				Name  string `db:"name"`
				NoTag string
			}{
				Name:  "test",
				NoTag: "ignored",
			},
			wantKeys: []string{"name"},
			checkFunc: func(t *testing.T, result map[string]any) {
				if _, exists := result["NoTag"]; exists {
					t.Errorf("NoTag field should be skipped")
				}
			},
		},
		// Nice to have: Empty struct test
		{
			name: "handles empty struct",
			input: &struct {
			}{},
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.input)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}

			result, err := prepareFieldsFromStruct(val)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("prepareFieldsFromStruct() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If we expected an error, we're done
			if tt.wantErr {
				return
			}

			if len(result) != len(tt.wantKeys) {
				t.Errorf("got %d fields, want %d", len(result), len(tt.wantKeys))
			}

			for _, key := range tt.wantKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("missing key: %s", key)
				}
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestPrepareFieldsFromMap(t *testing.T) {
	id := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	tests := []struct {
		name      string
		input     any // Changed from map[string]any to support error tests
		wantErr   bool
		checkFunc func(t *testing.T, result map[string]any)
	}{
		{
			name: "primitives pass through",
			input: map[string]any{
				"name":  "test",
				"count": 42,
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				if result["name"] != "test" {
					t.Errorf("name = %v, want test", result["name"])
				}
				if result["count"] != 42 {
					t.Errorf("count = %v, want 42", result["count"])
				}
			},
		},
		{
			name: "UUID auto-detected and converted",
			input: map[string]any{
				"id": id,
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["id"].([]byte)
				if !ok {
					t.Errorf("id type = %T, want []byte", result["id"])
				}
				if len(bytes) != 16 {
					t.Errorf("id length = %d, want 16", len(bytes))
				}
			},
		},
		{
			name: "slice auto-marshaled to JSON",
			input: map[string]any{
				"tags": []string{"a", "b"},
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["tags"].([]byte)
				if !ok {
					t.Errorf("tags type = %T, want []byte", result["tags"])
				}
				// Verify JSON content
				if string(bytes) != `["a","b"]` {
					t.Errorf("tags = %s, want [\"a\",\"b\"]", string(bytes))
				}
			},
		},
		{
			name: "nil value preserved",
			input: map[string]any{
				"optional": nil,
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				if result["optional"] != nil {
					t.Errorf("optional = %v, want nil", result["optional"])
				}
			},
		},
		// Edge case: Empty map
		{
			name:  "empty map",
			input: map[string]any{},
			checkFunc: func(t *testing.T, result map[string]any) {
				if len(result) != 0 {
					t.Errorf("expected empty map, got %d fields", len(result))
				}
			},
		},
		// Edge case: Map type conversion
		{
			name: "map marshaled to JSON",
			input: map[string]any{
				"config": map[string]int{"timeout": 30, "retries": 3},
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["config"].([]byte)
				if !ok {
					t.Errorf("config type = %T, want []byte", result["config"])
				}
				// Verify it's valid JSON
				if len(bytes) == 0 {
					t.Errorf("config should not be empty")
				}
			},
		},
		// Edge case: Struct type conversion
		{
			name: "struct marshaled to JSON",
			input: map[string]any{
				"metadata": struct {
					Version string
					Build   int
				}{Version: "1.0", Build: 123},
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				bytes, ok := result["metadata"].([]byte)
				if !ok {
					t.Errorf("metadata type = %T, want []byte", result["metadata"])
				}
				expected := `{"Version":"1.0","Build":123}`
				if string(bytes) != expected {
					t.Errorf("metadata = %s, want %s", string(bytes), expected)
				}
			},
		},
		// Error path: Invalid input type
		{
			name:    "error on invalid input type",
			input:   "not a map",
			wantErr: true,
		},
		// Error path: Unmarshalable value (struct with channel can't be JSON marshaled)
		{
			name: "error on unmarshalable value",
			input: map[string]any{
				"badstruct": struct {
					Ch chan int
				}{Ch: make(chan int)},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareFieldsFromMap(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("prepareFieldsFromMap() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If we expected an error, we're done
			if tt.wantErr {
				return
			}

			// Verify result length matches input length
			if inputMap, ok := tt.input.(map[string]any); ok {
				if len(result) != len(inputMap) {
					t.Errorf("got %d fields, want %d", len(result), len(inputMap))
				}
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestPrepareFields(t *testing.T) {
	id := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	t.Run("accepts struct", func(t *testing.T) {
		input := &testModel{
			ID:    id,
			Name:  "test",
			Count: 42,
		}

		result, err := PrepareFields(input)
		if err != nil {
			t.Fatalf("PrepareFields() error = %v", err)
		}

		if len(result) != 3 { // id, name, count (SkipMe excluded)
			t.Errorf("got %d fields, want 3", len(result))
		}

		if _, exists := result["id"]; !exists {
			t.Errorf("missing id field")
		}
		if result["name"] != "test" {
			t.Errorf("name = %v, want test", result["name"])
		}
		if result["count"] != 42 {
			t.Errorf("count = %v, want 42", result["count"])
		}
	})

	t.Run("accepts map", func(t *testing.T) {
		input := map[string]any{
			"name":  "updated",
			"count": 100,
		}

		result, err := PrepareFields(input)
		if err != nil {
			t.Fatalf("PrepareFields() error = %v", err)
		}

		if len(result) != 2 {
			t.Errorf("got %d fields, want 2", len(result))
		}

		if result["name"] != "updated" {
			t.Errorf("name = %v, want updated", result["name"])
		}
	})

	t.Run("rejects invalid type", func(t *testing.T) {
		input := []string{"not", "supported"}

		_, err := PrepareFields(input)
		if err == nil {
			t.Errorf("PrepareFields() expected error for slice input")
		}
	})
}

func TestScanRow_Basic(t *testing.T) {
	// Note: Full tests will be copied from dbutil/main_test.go
	// This is a minimal test to verify the function exists

	t.Run("rejects non-pointer", func(t *testing.T) {
		// Create mock rows (simplified - full implementation uses test helper)
		var dest testModel
		err := ScanRow(nil, dest) // Non-pointer

		if err == nil {
			t.Errorf("ScanRow should reject non-pointer destination")
		}
	})
}

func TestConvertValueForDB(t *testing.T) {
	testUUID := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))
	expectedUUIDBytes := testUUID.Bytes()

	tests := []struct {
		name           string
		value          interface{}
		conversionType string
		wantType       string // Expected type name
		wantValue      interface{} // Expected value for verification
		wantErr        bool
	}{
		// UUID conversions
		{
			name:           "UUID to bytes",
			value:          testUUID,
			conversionType: "uuid",
			wantType:       "[]uint8",
			wantValue:      expectedUUIDBytes,
			wantErr:        false,
		},
		{
			name:           "invalid type for UUID conversion",
			value:          "not-a-uuid",
			conversionType: "uuid",
			wantType:       "",
			wantValue:      nil,
			wantErr:        true,
		},
		{
			name:           "UUID passthrough without conversion type",
			value:          testUUID,
			conversionType: "",
			wantType:       "uuid.UUID",
			wantValue:      testUUID,
			wantErr:        false,
		},
		// JSON conversions
		{
			name:           "slice to JSON",
			value:          []string{"a", "b", "c"},
			conversionType: "json",
			wantType:       "[]uint8",
			wantValue:      []byte(`["a","b","c"]`),
			wantErr:        false,
		},
		{
			name:           "map to JSON via explicit conversion",
			value:          map[string]int{"key": 123},
			conversionType: "json",
			wantType:       "[]uint8",
			wantValue:      []byte(`{"key":123}`),
			wantErr:        false,
		},
		// Auto-detection for complex types
		{
			name:           "map auto-marshaled to JSON",
			value:          map[string]int{"key": 123},
			conversionType: "",
			wantType:       "[]uint8",
			wantValue:      []byte(`{"key":123}`),
			wantErr:        false,
		},
		{
			name:           "struct auto-marshaled to JSON",
			value:          struct{ Name string }{"test"},
			conversionType: "",
			wantType:       "[]uint8",
			wantValue:      []byte(`{"Name":"test"}`),
			wantErr:        false,
		},
		{
			name:           "slice auto-marshaled to JSON",
			value:          []int{1, 2, 3},
			conversionType: "",
			wantType:       "[]uint8",
			wantValue:      []byte(`[1,2,3]`),
			wantErr:        false,
		},
		// Primitive passthroughs
		{
			name:           "string passthrough",
			value:          "test string",
			conversionType: "",
			wantType:       "string",
			wantValue:      "test string",
			wantErr:        false,
		},
		{
			name:           "int passthrough",
			value:          42,
			conversionType: "",
			wantType:       "int",
			wantValue:      42,
			wantErr:        false,
		},
		{
			name:           "float passthrough",
			value:          3.14,
			conversionType: "",
			wantType:       "float64",
			wantValue:      3.14,
			wantErr:        false,
		},
		{
			name:           "bool passthrough",
			value:          true,
			conversionType: "",
			wantType:       "bool",
			wantValue:      true,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertValueForDB(tt.value, tt.conversionType)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertValueForDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify type
				gotType := reflect.TypeOf(result).String()
				if gotType != tt.wantType {
					t.Errorf("convertValueForDB() type = %v, want %v", gotType, tt.wantType)
				}

				// Verify value when provided
				if tt.wantValue != nil {
					if !reflect.DeepEqual(result, tt.wantValue) {
						t.Errorf("convertValueForDB() value = %v, want %v", result, tt.wantValue)
					}
				}
			}
		})
	}
}
