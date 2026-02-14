package number

import (
	"testing"
)

func TestValidateVirtualNumber(t *testing.T) {
	tests := []struct {
		name         string
		num          string
		allowReserved bool
		expectErr    bool
	}{
		// valid cases
		{"valid virtual number", "+899001000001", false, false},
		{"valid virtual number max", "+899999999999", false, false},
		{"valid virtual number mid", "+899500123456", false, false},
		{"valid reserved with allow", "+899000000000", true, false},
		{"valid reserved max with allow", "+899000999999", true, false},

		// invalid prefix
		{"missing +899 prefix", "+15551234567", false, true},
		{"missing + sign", "899001000001", false, true},
		{"wrong prefix", "+998001000001", false, true},

		// invalid length
		{"too short", "+89900100000", false, true},
		{"too long", "+8990010000001", false, true},
		{"empty string", "", false, true},
		{"just prefix", "+899", false, true},

		// invalid characters
		{"contains letter", "+899001a00001", false, true},
		{"contains space", "+899001 00001", false, true},
		{"contains dash", "+899-01000001", false, true},

		// reserved range
		{"reserved range rejected", "+899000000000", false, true},
		{"reserved range max rejected", "+899000999999", false, true},
		{"reserved range mid rejected", "+899000500000", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVirtualNumber(tt.num, tt.allowReserved)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_normal", TypeNormal, "normal"},
		{"type_virtual", TypeVirtual, "virtual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
