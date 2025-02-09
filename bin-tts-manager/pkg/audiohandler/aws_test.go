package audiohandler

import (
	"testing"

	"monorepo/bin-tts-manager/models/tts"
)

func Test_awsGetVoiceID(t *testing.T) {
	handler := &audioHandler{}

	tests := []struct {
		lang     string
		gender   tts.Gender
		expected string
	}{
		{"en-US", tts.GenderFemale, "Joanna"},
		{"en-US", tts.GenderMale, "Matthew"},
		{"en-US", tts.GenderNeutral, "Joey"},
		{"en-GB", tts.GenderFemale, "Amy"},
		{"en-GB", tts.GenderMale, "Brian"},
		{"en-GB", tts.GenderNeutral, "Emma"},
		{"de-DE", tts.GenderFemale, "Marlene"},
		{"de-DE", tts.GenderMale, "Hans"},
		{"de-DE", tts.GenderNeutral, "Vicki"},
		{"fr-FR", tts.GenderFemale, "Celine"},
		{"fr-FR", tts.GenderMale, "Mathieu"},
		{"fr-FR", tts.GenderNeutral, "Lea"},
		{"es-ES", tts.GenderFemale, "Conchita"},
		{"es-ES", tts.GenderMale, "Enrique"},
		{"es-ES", tts.GenderNeutral, "Lucia"},
		{"it-IT", tts.GenderFemale, "Carla"},
		{"it-IT", tts.GenderMale, "Giorgio"},
		{"it-IT", tts.GenderNeutral, "Bianca"},
		{"ja-JP", tts.GenderFemale, "Mizuki"},
		{"ja-JP", tts.GenderMale, "Takumi"},
		{"ko-KR", tts.GenderFemale, "Seoyeon"},
		{"ko-KR", tts.GenderNeutral, "Jisoo"},
		{"unknown", tts.GenderFemale, ""},
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
