package audiohandler

import (
	"reflect"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

func Test_gcpGetDefaultVoiceName(t *testing.T) {

	type test struct {
		name string

		lang string

		expectRes string
	}

	tests := []test{
		{
			name:      "en-US",
			lang:      "en-US",
			expectRes: "en-US-Wavenet-F",
		},
		{
			name:      "en-GB",
			lang:      "en-GB",
			expectRes: "en-GB-Wavenet-A",
		},
		{
			name:      "de-DE",
			lang:      "de-DE",
			expectRes: "de-DE-Wavenet-F",
		},
		{
			name:      "fr-FR",
			lang:      "fr-FR",
			expectRes: "fr-FR-Wavenet-E",
		},
		{
			name:      "es-ES",
			lang:      "es-ES",
			expectRes: "es-ES-Wavenet-E",
		},
		{
			name:      "it-IT",
			lang:      "it-IT",
			expectRes: "it-IT-Wavenet-E",
		},
		{
			name:      "ja-JP",
			lang:      "ja-JP",
			expectRes: "ja-JP-Wavenet-C",
		},
		{
			name:      "ko-KR",
			lang:      "ko-KR",
			expectRes: "ko-KR-Wavenet-C",
		},
		{
			name:      "unknown language",
			lang:      "unknown",
			expectRes: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &audioHandler{}

			res := h.gcpGetDefaultVoiceName(tt.lang)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}
