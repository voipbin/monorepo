package audiohandler

import (
	"testing"

	"monorepo/bin-tts-manager/models/tts"

	"github.com/aws/aws-sdk-go-v2/service/polly/types"
)

func Test_awsGetVoiceID(t *testing.T) {
	handler := &audioHandler{}

	tests := []struct {
		name     string
		lang     string
		gender   tts.Gender
		expected types.VoiceId
	}{
		{
			name:     "US Female",
			lang:     "en-US",
			gender:   tts.GenderFemale,
			expected: "Joanna",
		},
		{
			name:     "US Male",
			lang:     "en-US",
			gender:   tts.GenderMale,
			expected: "Matthew",
		},
		{
			name:     "US Neutral",
			lang:     "en-US",
			gender:   tts.GenderNeutral,
			expected: "Joey",
		},
		{
			name:     "GB Female",
			lang:     "en-GB",
			gender:   tts.GenderFemale,
			expected: "Amy",
		},
		{
			name:     "GB Male",
			lang:     "en-GB",
			gender:   tts.GenderMale,
			expected: "Brian",
		},
		{
			name:     "GB Neutral",
			lang:     "en-GB",
			gender:   tts.GenderNeutral,
			expected: "Emma",
		},
		{
			name:     "DE Female",
			lang:     "de-DE",
			gender:   tts.GenderFemale,
			expected: "Marlene",
		},
		{
			name:     "DE Male",
			lang:     "de-DE",
			gender:   tts.GenderMale,
			expected: "Hans",
		},
		{
			name:     "DE Neutral",
			lang:     "de-DE",
			gender:   tts.GenderNeutral,
			expected: "Vicki",
		},
		{
			name:     "FR Female",
			lang:     "fr-FR",
			gender:   tts.GenderFemale,
			expected: "Celine",
		},
		{
			name:     "FR Male",
			lang:     "fr-FR",
			gender:   tts.GenderMale,
			expected: "Mathieu",
		},
		{
			name:     "FR Neutral",
			lang:     "fr-FR",
			gender:   tts.GenderNeutral,
			expected: "Lea",
		},
		{
			name:     "ES Female",
			lang:     "es-ES",
			gender:   tts.GenderFemale,
			expected: "Conchita",
		},
		{
			name:     "ES Male",
			lang:     "es-ES",
			gender:   tts.GenderMale,
			expected: "Enrique",
		},
		{
			name:     "ES Neutral",
			lang:     "es-ES",
			gender:   tts.GenderNeutral,
			expected: "Lucia",
		},
		{
			name:     "IT Female",
			lang:     "it-IT",
			gender:   tts.GenderFemale,
			expected: "Carla",
		},
		{
			name:     "IT Male",
			lang:     "it-IT",
			gender:   tts.GenderMale,
			expected: "Giorgio",
		},
		{
			name:     "IT Neutral",
			lang:     "it-IT",
			gender:   tts.GenderNeutral,
			expected: "Bianca",
		},
		{
			name:     "JP Female",
			lang:     "ja-JP",
			gender:   tts.GenderFemale,
			expected: "Mizuki",
		},
		{
			name:     "JP Male",
			lang:     "ja-JP",
			gender:   tts.GenderMale,
			expected: "Takumi",
		},
		{
			name:     "KR Female",
			lang:     "ko-KR",
			gender:   tts.GenderFemale,
			expected: "Seoyeon",
		},
		{
			name:     "KR Neutral",
			lang:     "ko-KR",
			gender:   tts.GenderNeutral,
			expected: "Jihye",
		},
		{
			name:     "BR Female",
			lang:     "pt-BR",
			gender:   tts.GenderFemale,
			expected: "Camila",
		},
		{
			name:     "BR Male",
			lang:     "pt-BR",
			gender:   tts.GenderMale,
			expected: "Ricardo",
		},
		{
			name:     "BR Neutral",
			lang:     "pt-BR",
			gender:   tts.GenderNeutral,
			expected: "Camila",
		},
		{
			name:     "RU Female",
			lang:     "ru-RU",
			gender:   tts.GenderFemale,
			expected: "Tatyana",
		},
		{
			name:     "RU Male",
			lang:     "ru-RU",
			gender:   tts.GenderMale,
			expected: "Maxim",
		},
		{
			name:     "RU Neutral",
			lang:     "ru-RU",
			gender:   tts.GenderNeutral,
			expected: "Tatyana",
		},
		{
			name:     "CN Female",
			lang:     "zh-CN",
			gender:   tts.GenderFemale,
			expected: "Zhiyu",
		},
		{
			name:     "CN Male",
			lang:     "zh-CN",
			gender:   tts.GenderMale,
			expected: "Zhiyu",
		},
		{
			name:     "CN Neutral",
			lang:     "zh-CN",
			gender:   tts.GenderNeutral,
			expected: "Zhiyu",
		},
		{
			name:     "Unknown Language",
			lang:     "unknown",
			gender:   tts.GenderFemale,
			expected: "",
		},
		{
			name:     "Unknown Gender",
			lang:     "en-US",
			gender:   tts.Gender("unknown"),
			expected: "",
		},
		{
			name:     "Unknown Language and Gender",
			lang:     "unknown",
			gender:   tts.Gender("unknown"),
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.lang+"_"+string(test.gender), func(t *testing.T) {
			result := handler.awsGetVoiceID(test.lang, test.gender)
			if result != test.expected {
				t.Errorf("expected %s, got %s", test.expected, result)
			}
		})
	}
}
