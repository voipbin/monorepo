package audiohandler

import (
	"reflect"
	"testing"

	"monorepo/bin-tts-manager/models/tts"

	gomock "go.uber.org/mock/gomock"
)

func Test_gcpGetVoiceName(t *testing.T) {

	type test struct {
		name string

		lang   string
		gender tts.Gender

		expectRes string
	}

	tests := []test{
		{
			name:      "en-US female",
			lang:      "en-US",
			gender:    tts.GenderFemale,
			expectRes: "en-US-Wavenet-F",
		},
		{
			name:      "en-US male",
			lang:      "en-US",
			gender:    tts.GenderMale,
			expectRes: "en-US-Wavenet-D",
		},
		{
			name:      "en-GB neutral",
			lang:      "en-GB",
			gender:    tts.GenderNeutral,
			expectRes: "en-GB-Wavenet-D",
		},
		{
			name:      "de-DE female",
			lang:      "de-DE",
			gender:    tts.GenderFemale,
			expectRes: "de-DE-Wavenet-F",
		},
		{
			name:      "fr-FR male",
			lang:      "fr-FR",
			gender:    tts.GenderMale,
			expectRes: "fr-FR-Wavenet-B",
		},
		{
			name:      "es-ES neutral",
			lang:      "es-ES",
			gender:    tts.GenderNeutral,
			expectRes: "es-ES-Wavenet-A",
		},
		{
			name:      "it-IT female",
			lang:      "it-IT",
			gender:    tts.GenderFemale,
			expectRes: "it-IT-Wavenet-E",
		},
		{
			name:      "ja-JP male",
			lang:      "ja-JP",
			gender:    tts.GenderMale,
			expectRes: "ja-JP-Wavenet-B",
		},
		{
			name:      "ko-KR neutral",
			lang:      "ko-KR",
			gender:    tts.GenderNeutral,
			expectRes: "ko-KR-Wavenet-A",
		},
		{
			name:      "unknown language",
			lang:      "unknown",
			gender:    tts.GenderNeutral,
			expectRes: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &audioHandler{}

			res := h.gcpGetVoiceName(tt.lang, tt.gender)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}
