package transcribehandler

import "testing"

func Test_getBCP47LanguageCode_fallback(t *testing.T) {
	tests := []struct {
		name string

		language  string
		expectRes string
	}{
		{
			name:      "an unparsable language falls back to the default",
			language:  "zz-invalid",
			expectRes: "en-US",
		},
		{
			name:      "an empty language falls back to the default",
			language:  "",
			expectRes: "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := getBCP47LanguageCode(tt.language); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}
