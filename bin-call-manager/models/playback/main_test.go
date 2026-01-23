package playback

import (
	"testing"
)

func TestIDPrefixConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"id_prefix_call", IDPrefixCall, "call:"},
		{"id_prefix_external_media", IDPrefixExternalMedia, "externalmedia:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
