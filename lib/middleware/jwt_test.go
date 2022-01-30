package middleware

import (
	"testing"
)

func TestGenerateToken(t *testing.T) {

	tests := []struct {
		name string

		key  string
		data string
	}{
		{
			"normal",

			"test",
			"test22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := GenerateToken(tt.key, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tmp, err := validateToken(res)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if tmp[tt.key] != tt.data {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.data, tmp[tt.key])
			}
		})
	}
}
