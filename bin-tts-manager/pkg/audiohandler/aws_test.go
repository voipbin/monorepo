package audiohandler

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/polly/types"
)

func Test_awsGetDefaultVoiceID(t *testing.T) {
	handler := &audioHandler{}

	tests := []struct {
		name     string
		lang     string
		expected types.VoiceId
	}{
		{
			name:     "en-US",
			lang:     "en-US",
			expected: types.VoiceIdJoanna,
		},
		{
			name:     "en-GB",
			lang:     "en-GB",
			expected: types.VoiceIdAmy,
		},
		{
			name:     "de-DE",
			lang:     "de-DE",
			expected: types.VoiceIdMarlene,
		},
		{
			name:     "fr-FR",
			lang:     "fr-FR",
			expected: types.VoiceIdCeline,
		},
		{
			name:     "es-ES",
			lang:     "es-ES",
			expected: types.VoiceIdConchita,
		},
		{
			name:     "it-IT",
			lang:     "it-IT",
			expected: types.VoiceIdCarla,
		},
		{
			name:     "ja-JP",
			lang:     "ja-JP",
			expected: types.VoiceIdMizuki,
		},
		{
			name:     "ko-KR",
			lang:     "ko-KR",
			expected: types.VoiceIdSeoyeon,
		},
		{
			name:     "pt-BR",
			lang:     "pt-BR",
			expected: types.VoiceIdCamila,
		},
		{
			name:     "ru-RU",
			lang:     "ru-RU",
			expected: types.VoiceIdTatyana,
		},
		{
			name:     "zh-CN",
			lang:     "zh-CN",
			expected: types.VoiceIdZhiyu,
		},
		{
			name:     "Unknown Language",
			lang:     "unknown",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := handler.awsGetDefaultVoiceID(test.lang)
			if result != test.expected {
				t.Errorf("expected %s, got %s", test.expected, result)
			}
		})
	}
}
