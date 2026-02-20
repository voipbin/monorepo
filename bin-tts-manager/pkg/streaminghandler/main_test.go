package streaminghandler

import "testing"

func Test_formatForProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"gcp", "gcp", formatUlaw},
		{"aws", "aws", formatSlin},
		{"elevenlabs", "elevenlabs", formatSlin16},
		{"empty defaults to slin16", "", formatSlin16},
		{"unknown defaults to slin16", "unknown_vendor", formatSlin16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatForProvider(tt.provider)
			if got != tt.expected {
				t.Errorf("formatForProvider(%q) = %q, want %q", tt.provider, got, tt.expected)
			}
		})
	}
}
