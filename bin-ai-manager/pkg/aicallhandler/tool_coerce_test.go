package aicallhandler

import (
	"encoding/json"
	"testing"
)

func Test_coerceToolVariables(t *testing.T) {
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
			name: "number coerced to string without trailing dot-zero",
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
			name: "all non-scalar returns nil",
			in:     map[string]any{"obj": map[string]any{"a": 1}, "null": nil},
			expect: nil,
		},
		{
			name: "json.Number stringifies exactly without float precision loss",
			in:   map[string]any{"big_id": json.Number("123456789012345678")},
			expect: map[string]string{
				"big_id": "123456789012345678",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coerceToolVariables(tt.in)

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
