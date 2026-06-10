package server

import (
	"strings"
	"testing"
)

func Test_validateVariables(t *testing.T) {
	// build helpers
	manyKeys := func(n int) map[string]string {
		m := make(map[string]string, n)
		for i := 0; i < n; i++ {
			m["k"+itoa(i)] = "v"
		}
		return m
	}

	tests := []struct {
		name string

		variables map[string]string

		expectError      bool
		expectMsgSubstr  string
	}{
		{
			name:        "nil map returns nil",
			variables:   nil,
			expectError: false,
		},
		{
			name:        "empty map returns nil",
			variables:   map[string]string{},
			expectError: false,
		},
		{
			name:        "small map under limits returns nil",
			variables:   map[string]string{"a": "1", "b": "2", "c": "3"},
			expectError: false,
		},
		{
			name:        "exactly 100 keys returns nil",
			variables:   manyKeys(100),
			expectError: false,
		},
		{
			name:            "101 keys exceeds max keys",
			variables:       manyKeys(101),
			expectError:     true,
			expectMsgSubstr: "too many variables (max 100)",
		},
		{
			name: "single value over 32KB",
			variables: map[string]string{
				"big": strings.Repeat("x", 32*1024+1),
			},
			expectError:     true,
			expectMsgSubstr: "variable value exceeds 32KB",
		},
		{
			name:            "total over 64KB with each value under 32KB",
			variables:       overTotalVariables(),
			expectError:     true,
			expectMsgSubstr: "variables total size exceeds 64KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := validateVariables(tt.variables)

			if !tt.expectError {
				if res != nil {
					t.Errorf("expected nil error, got %v", res)
				}
				return
			}

			if res == nil {
				t.Fatalf("expected non-nil error, got nil")
			}
			if !strings.Contains(res.Message, tt.expectMsgSubstr) {
				t.Errorf("expected message to contain %q, got %q", tt.expectMsgSubstr, res.Message)
			}
		})
	}
}

func Test_convertVariables(t *testing.T) {
	t.Run("nil pointer returns nil", func(t *testing.T) {
		if got := convertVariables(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("non-nil pointer returns dereferenced map", func(t *testing.T) {
		in := map[string]string{"a": "1", "b": "2"}
		got := convertVariables(&in)
		if len(got) != len(in) {
			t.Fatalf("length mismatch. expect %d, got %d", len(in), len(got))
		}
		for k, v := range in {
			if got[k] != v {
				t.Errorf("key %q: expect %q, got %q", k, v, got[k])
			}
		}
	})
}

// overTotalVariables builds a map where every individual value is under 32KB
// but the sum of len(key)+len(value) exceeds 64KB.
func overTotalVariables() map[string]string {
	// 4 values of 20KB each = 80KB total, each value well under the 32KB single cap.
	m := make(map[string]string, 4)
	for i := 0; i < 4; i++ {
		m["k"+itoa(i)] = strings.Repeat("y", 20*1024)
	}
	return m
}

// itoa is a tiny dependency-free int-to-string helper for test key generation.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
