package requesthandler

import (
	"monorepo/bin-common-handler/models/sock"
	"reflect"
	"testing"
)

func Test_GetFilteredItems(t *testing.T) {
	tests := []struct {
		name string

		m       *sock.Request
		filters []string

		expectRes  map[string]any
		shouldFail bool
	}{
		{
			name: "valid request with matching filters",
			m: &sock.Request{
				Data: []byte(`{
					"name": "Alice",
					"age": 30,
					"email": "alice@example.com"
				}`),
			},
			filters: []string{"name", "email", "age"},

			expectRes: map[string]any{
				"name":  "Alice",
				"age":   float64(30),
				"email": "alice@example.com",
			},
			shouldFail: false,
		},
		{
			name: "valid request with no matching filters",
			m: &sock.Request{
				Data: []byte(`{
					"foo": "bar",
					"id": 42
				}`),
			},
			filters: []string{"name"},

			expectRes:  map[string]any{},
			shouldFail: false,
		},
		{
			name: "invalid JSON",
			m: &sock.Request{
				Data: []byte(`{invalid-json}`),
			},
			filters: []string{"name"},

			expectRes:  nil,
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFilteredItems(tt.m, tt.filters)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if got != nil {
					t.Errorf("expected nil result, got %+v", got)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tt.expectRes) {
					t.Errorf("expected %+v, got %+v", tt.expectRes, got)
				}
			}
		})
	}
}
