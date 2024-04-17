package transcribehandler

import "testing"

func TestGetBCP47LanguageCode(t *testing.T) {
	tests := []struct {
		name string

		language  string
		expectRes string
	}{
		{
			"english us",

			"en-US",
			"en-US",
		},
		{
			"korean korea",

			"ko-KR",
			"ko-KR",
		},
		{
			"japanese japan",

			"ja-JP",
			"ja-JP",
		},
		{
			"Belgian Dutch",

			"nl-BE",
			"nl-BE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := getBCP47LanguageCode(tt.language)

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}
