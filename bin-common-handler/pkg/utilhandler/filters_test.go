package utilhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_ParseFiltersFromRequestBody(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expectRes map[string]any
		expectErr bool
	}{
		{
			"empty data",
			[]byte{},
			map[string]any{},
			false,
		},
		{
			"valid json with multiple types",
			[]byte(`{"customer_id":"5fd7f9b8-cb37-11ee-bd29-f30560a6ac86","deleted":false,"name":"test","count":42}`),
			map[string]any{
				"customer_id": "5fd7f9b8-cb37-11ee-bd29-f30560a6ac86",
				"deleted":     false,
				"name":        "test",
				"count":       float64(42), // JSON numbers are float64
			},
			false,
		},
		{
			"invalid json",
			[]byte(`{invalid json}`),
			nil,
			true,
		},
		{
			"empty json object",
			[]byte(`{}`),
			map[string]any{},
			false,
		},
		{
			"nested structures",
			[]byte(`{"customer_id":"test","metadata":{"key":"value"}}`),
			map[string]any{
				"customer_id": "test",
				"metadata":    map[string]any{"key": "value"},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseFiltersFromRequestBody(tt.data)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(res) != len(tt.expectRes) {
				t.Errorf("result length mismatch: got %d, want %d", len(res), len(tt.expectRes))
				return
			}

			for k, expectedVal := range tt.expectRes {
				gotVal, exists := res[k]
				if !exists {
					t.Errorf("expected key %s not found in result", k)
					continue
				}

				// Compare values based on type
				switch ev := expectedVal.(type) {
				case map[string]any:
					gv, ok := gotVal.(map[string]any)
					if !ok {
						t.Errorf("key %s: expected map[string]any, got %T", k, gotVal)
						continue
					}
					for mk, mv := range ev {
						if gv[mk] != mv {
							t.Errorf("key %s.%s: got %v, want %v", k, mk, gv[mk], mv)
						}
					}
				default:
					if gotVal != expectedVal {
						t.Errorf("key %s: got %v, want %v", k, gotVal, expectedVal)
					}
				}
			}
		})
	}
}

func Test_ConvertFilters(t *testing.T) {
	// Test FieldStruct with various types
	type TestFieldStruct struct {
		CustomerID uuid.UUID `filter:"customer_id"`
		Name       string    `filter:"name"`
		Deleted    bool      `filter:"deleted"`
		Count      int64     `filter:"count"`
		Ignored    string    `filter:"-"`
		NoTag      string
	}

	type TestField string

	tests := []struct {
		name      string
		filters   map[string]any
		expectRes map[TestField]any
		expectErr bool
	}{
		{
			"uuid string conversion",
			map[string]any{
				"customer_id": "5fd7f9b8-cb37-11ee-bd29-f30560a6ac86",
			},
			map[TestField]any{
				TestField("customer_id"): uuid.FromStringOrNil("5fd7f9b8-cb37-11ee-bd29-f30560a6ac86"),
			},
			false,
		},
		{
			"bool passthrough",
			map[string]any{
				"deleted": true,
			},
			map[TestField]any{
				TestField("deleted"): true,
			},
			false,
		},
		{
			"string passthrough",
			map[string]any{
				"name": "test-name",
			},
			map[TestField]any{
				TestField("name"): "test-name",
			},
			false,
		},
		{
			"number conversion (float64 to int64)",
			map[string]any{
				"count": float64(42),
			},
			map[TestField]any{
				TestField("count"): int64(42),
			},
			false,
		},
		{
			"multiple filters",
			map[string]any{
				"customer_id": "5fd7f9b8-cb37-11ee-bd29-f30560a6ac86",
				"name":        "test",
				"deleted":     false,
				"count":       float64(100),
			},
			map[TestField]any{
				TestField("customer_id"): uuid.FromStringOrNil("5fd7f9b8-cb37-11ee-bd29-f30560a6ac86"),
				TestField("name"):        "test",
				TestField("deleted"):     false,
				TestField("count"):       int64(100),
			},
			false,
		},
		{
			"unknown filter key ignored",
			map[string]any{
				"unknown_field": "should be ignored",
				"name":          "test",
			},
			map[TestField]any{
				TestField("name"): "test",
			},
			false,
		},
		{
			"filter with dash tag ignored",
			map[string]any{
				"ignored": "should not appear",
				"name":    "test",
			},
			map[TestField]any{
				TestField("name"): "test",
			},
			false,
		},
		{
			"field without tag ignored",
			map[string]any{
				"no_tag": "should not appear",
				"name":   "test",
			},
			map[TestField]any{
				TestField("name"): "test",
			},
			false,
		},
		{
			"invalid uuid format",
			map[string]any{
				"customer_id": "invalid-uuid",
			},
			nil,
			true,
		},
		{
			"empty filters",
			map[string]any{},
			map[TestField]any{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ConvertFilters[TestFieldStruct, TestField](TestFieldStruct{}, tt.filters)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(res) != len(tt.expectRes) {
				t.Errorf("result length mismatch: got %d, want %d", len(res), len(tt.expectRes))
				return
			}

			for k, expectedVal := range tt.expectRes {
				gotVal, exists := res[k]
				if !exists {
					t.Errorf("expected key %s not found in result", k)
					continue
				}

				// Compare UUIDs specially
				if expectedUUID, ok := expectedVal.(uuid.UUID); ok {
					gotUUID, ok := gotVal.(uuid.UUID)
					if !ok {
						t.Errorf("key %s: expected uuid.UUID, got %T", k, gotVal)
						continue
					}
					if gotUUID != expectedUUID {
						t.Errorf("key %s: got %v, want %v", k, gotUUID, expectedUUID)
					}
				} else {
					if gotVal != expectedVal {
						t.Errorf("key %s: got %v (%T), want %v (%T)", k, gotVal, gotVal, expectedVal, expectedVal)
					}
				}
			}
		})
	}
}

func Test_convertValueToType(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		targetType string // Using string representation for easier test definition
		expectRes  any
		expectErr  bool
	}{
		{
			"nil value",
			nil,
			"string",
			nil,
			false,
		},
		{
			"uuid string to UUID",
			"5fd7f9b8-cb37-11ee-bd29-f30560a6ac86",
			"uuid.UUID",
			uuid.FromStringOrNil("5fd7f9b8-cb37-11ee-bd29-f30560a6ac86"),
			false,
		},
		{
			"invalid uuid string",
			"invalid-uuid-format",
			"uuid.UUID",
			nil,
			true,
		},
		{
			"bool to bool",
			true,
			"bool",
			true,
			false,
		},
		{
			"string to string",
			"test",
			"string",
			"test",
			false,
		},
		{
			"float64 to int64",
			float64(42),
			"int64",
			int64(42),
			false,
		},
		{
			"float64 overflow to int64",
			float64(1e20), // Exceeds int64 max
			"int64",
			nil,
			true,
		},
		{
			"negative float64 overflow to int64",
			float64(-1e20), // Exceeds int64 min
			"int64",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targetType any

			// Create target type based on string representation
			switch tt.targetType {
			case "uuid.UUID":
				var u uuid.UUID
				targetType = u
			case "bool":
				var b bool
				targetType = b
			case "string":
				var s string
				targetType = s
			case "int64":
				var i int64
				targetType = i
			}

			res, err := convertValueToType(tt.value, reflect.TypeOf(targetType))

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Special handling for UUID comparison
			if expectedUUID, ok := tt.expectRes.(uuid.UUID); ok {
				gotUUID, ok := res.(uuid.UUID)
				if !ok {
					t.Errorf("expected uuid.UUID, got %T", res)
					return
				}
				if gotUUID != expectedUUID {
					t.Errorf("got %v, want %v", gotUUID, expectedUUID)
				}
			} else {
				if res != tt.expectRes {
					t.Errorf("got %v (%T), want %v (%T)", res, res, tt.expectRes, tt.expectRes)
				}
			}
		})
	}
}
