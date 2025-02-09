package audiohandler

// func Test_gcpGetVoiceName(t *testing.T) {

// 	type test struct {
// 		name string

// 		lang   string
// 		gender tts.Gender

// 		expectRes string
// 	}

// 	tests := []test{
// 		{
// 			"en-US female",

// 			"en-US",
// 			tts.GenderFemale,

// 			"en-US-Wavenet-F",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			h := &audioHandler{}

// 			res := h.gcpGetVoiceName(tt.lang, tt.gender)
// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

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
			"en-US female",

			"en-US",
			tts.GenderFemale,

			"en-US-Wavenet-F",
		},
		{
			"en-US male",

			"en-US",
			tts.GenderMale,

			"en-US-Wavenet-D",
		},
		{
			"en-GB neutral",

			"en-GB",
			tts.GenderNeutral,

			"en-GB-Wavenet-D",
		},
		{
			"de-DE female",

			"de-DE",
			tts.GenderFemale,

			"de-DE-Wavenet-F",
		},
		{
			"fr-FR male",

			"fr-FR",
			tts.GenderMale,

			"fr-FR-Wavenet-B",
		},
		{
			"es-ES neutral",

			"es-ES",
			tts.GenderNeutral,

			"es-ES-Wavenet-A",
		},
		{
			"it-IT female",

			"it-IT",
			tts.GenderFemale,

			"it-IT-Wavenet-E",
		},
		{
			"ja-JP male",

			"ja-JP",
			tts.GenderMale,

			"ja-JP-Wavenet-B",
		},
		{
			"ko-KR neutral",

			"ko-KR",
			tts.GenderNeutral,

			"ko-KR-Wavenet-A",
		},
		{
			"unknown language",

			"unknown",
			tts.GenderNeutral,

			"",
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
