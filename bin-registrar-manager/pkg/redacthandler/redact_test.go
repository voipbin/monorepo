package redacthandler

import (
	"reflect"
	"testing"
)

type testField string

func Test_IsSensitive(t *testing.T) {

	tests := []struct {
		name string
		key  string

		expectRes bool
	}{
		{"password lowercase", "password", true},
		{"Password mixed case", "Password", true},
		{"SECRET uppercase", "SECRET", true},
		{"token", "token", true},
		{"credential", "credential", true},
		{"hash", "hash", true},
		{"direct_hash suffix match", "direct_hash", true},
		{"Direct_Hash mixed case", "Direct_Hash", true},
		{"unrelated key", "customer_id", false},
		{"unrelated key ending in ash but not hash", "trash", false},
		{"empty key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := IsSensitive(tt.key)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Fields(t *testing.T) {

	tests := []struct {
		name   string
		fields map[testField]any

		expectRes map[testField]any
	}{
		{
			name: "password field is redacted, others untouched",
			fields: map[testField]any{
				testField("name"):     "ext-1000",
				testField("password"): "plaintext-secret",
			},
			expectRes: map[testField]any{
				testField("name"):     "ext-1000",
				testField("password"): Placeholder,
			},
		},
		{
			name:      "empty map",
			fields:    map[testField]any{},
			expectRes: map[testField]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Fields(tt.fields)
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			// the original map passed in by the caller must be untouched
			if _, ok := tt.fields[testField("password")]; ok {
				if tt.fields[testField("password")] == Placeholder {
					t.Errorf("Original fields map was mutated: %v", tt.fields)
				}
			}
		})
	}
}

func Test_String(t *testing.T) {

	tests := []struct {
		name string
		s    string

		expectRes string
	}{
		{"non-empty value is masked", "plaintext-secret", Placeholder},
		{"empty value stays empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := String(tt.s)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}
