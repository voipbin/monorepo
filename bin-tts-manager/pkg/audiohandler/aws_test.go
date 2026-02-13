package audiohandler

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/gofrs/uuid"
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

func Test_awsAudioCreate_unknownLangNoVoiceID(t *testing.T) {
	h := &audioHandler{}
	ctx := context.Background()
	callID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")

	err := h.awsAudioCreate(ctx, callID, "<speak>hello</speak>", "xx-XX", "", "/tmp/test.wav")
	if err == nil {
		t.Error("expected error for unknown language with no voice_id, got nil")
	}

	expectedMsg := `no default voice available for language "xx-XX" and no voice_id provided`
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message.\nexpect: %s\ngot: %s", expectedMsg, err.Error())
	}
}
