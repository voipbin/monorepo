package outboundconfig

import (
	"strings"
	"testing"
)

func Test_ValidateCodecs_cov(t *testing.T) {
	tests := []struct {
		name   string
		codecs string
		expect bool
	}{
		{"empty is server default", "", true},
		{"single", "ulaw", true},
		{"multiple", "ulaw,alaw,g722", true},
		{"hyphenated tokens", "AMR-WB,GSM-EFR,telephone-event", true},
		{"exactly 255 chars", strings.Repeat("a", 255), true},
		{"256 chars", strings.Repeat("a", 256), false},
		{"newline", "ulaw\nalaw", false},
		{"carriage return", "ulaw\r", false},
		{"trailing newline", "ulaw\n", false},
		{"leading newline", "\nulaw", false},
		{"leading carriage return", "\rulaw", false},
		{"crlf", "ulaw\r\nalaw", false},
		{"leading hyphen", "-ulaw", false},
		{"trailing hyphen", "ulaw-", false},
		{"double hyphen", "u--law", false},
		{"empty token", "ulaw,,alaw", false},
		{"trailing comma", "ulaw,", false},
		{"space", "ulaw, alaw", false},
		{"special char", "ulaw;alaw", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := ValidateCodecs(tt.codecs); res != tt.expect {
				t.Errorf("Wrong match. codecs: %q, expect: %v, got: %v", tt.codecs, tt.expect, res)
			}
		})
	}
}
