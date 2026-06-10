package variable

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
)

func TestVariableStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	variables := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	v := Variable{
		ID:        id,
		Variables: variables,
	}

	if v.ID != id {
		t.Errorf("Variable.ID = %v, expected %v", v.ID, id)
	}
	if len(v.Variables) != 2 {
		t.Errorf("Variable.Variables length = %v, expected %v", len(v.Variables), 2)
	}
	if v.Variables["key1"] != "value1" {
		t.Errorf("Variable.Variables[key1] = %v, expected %v", v.Variables["key1"], "value1")
	}
	if v.Variables["key2"] != "value2" {
		t.Errorf("Variable.Variables[key2] = %v, expected %v", v.Variables["key2"], "value2")
	}
}

func TestVariableWithNilVariables(t *testing.T) {
	v := Variable{
		ID: uuid.Must(uuid.NewV4()),
	}

	if v.Variables != nil {
		t.Errorf("Variable.Variables should be nil, got %v", v.Variables)
	}
}

func TestVariableWithEmptyVariables(t *testing.T) {
	v := Variable{
		ID:        uuid.Must(uuid.NewV4()),
		Variables: map[string]string{},
	}

	if len(v.Variables) != 0 {
		t.Errorf("Variable.Variables length = %v, expected %v", len(v.Variables), 0)
	}
}

func TestVariableModification(t *testing.T) {
	v := Variable{
		ID:        uuid.Must(uuid.NewV4()),
		Variables: map[string]string{"existing": "value"},
	}

	v.Variables["new_key"] = "new_value"

	if len(v.Variables) != 2 {
		t.Errorf("Variable.Variables length = %v, expected %v after adding", len(v.Variables), 2)
	}
	if v.Variables["new_key"] != "new_value" {
		t.Errorf("Variable.Variables[new_key] = %v, expected %v", v.Variables["new_key"], "new_value")
	}
}

func Test_NewVariablesFromMap(t *testing.T) {
	tests := []struct {
		name string

		in map[string]any

		expect map[string]string
	}{
		{
			name:   "nil input returns nil",
			in:     nil,
			expect: nil,
		},
		{
			name:   "empty input returns nil",
			in:     map[string]any{},
			expect: nil,
		},
		{
			name: "string values pass through",
			in:   map[string]any{"campaign_id": "summer", "intent": "renewal"},
			expect: map[string]string{
				"campaign_id": "summer",
				"intent":      "renewal",
			},
		},
		{
			name: "float64 coerced without trailing dot-zero",
			in:   map[string]any{"count": float64(123), "ratio": float64(1.5)},
			expect: map[string]string{
				"count": "123",
				"ratio": "1.5",
			},
		},
		{
			name: "bool coerced to string",
			in:   map[string]any{"flag": true, "off": false},
			expect: map[string]string{
				"flag": "true",
				"off":  "false",
			},
		},
		{
			name: "json.Number stringifies exactly without float precision loss",
			in:   map[string]any{"big_id": json.Number("123456789012345678")},
			expect: map[string]string{
				"big_id": "123456789012345678",
			},
		},
		{
			name: "non-scalar values skipped, scalars kept",
			in: map[string]any{
				"good":   "v",
				"obj":    map[string]any{"a": 1},
				"arr":    []any{1, 2},
				"null":   nil,
				"number": float64(7),
			},
			expect: map[string]string{
				"good":   "v",
				"number": "7",
			},
		},
		{
			name:   "all non-scalar returns nil",
			in:     map[string]any{"obj": map[string]any{"a": 1}, "null": nil},
			expect: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewVariablesFromMap(tt.in)

			if tt.expect == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}

			if len(got) != len(tt.expect) {
				t.Errorf("length mismatch. expect %d, got %d (%v)", len(tt.expect), len(got), got)
			}
			for k, v := range tt.expect {
				if got[k] != v {
					t.Errorf("key %q: expect %q, got %q", k, v, got[k])
				}
			}
		})
	}
}
