package tts

import (
	"testing"
)

func TestTTS(t *testing.T) {
	tests := []struct {
		name string

		gender          Gender
		text            string
		language        string
		mediaBucketName string
		mediaFilepath   string
	}{
		{
			name: "creates_tts_with_all_fields",

			gender:          GenderMale,
			text:            "Hello, world!",
			language:        "en-US",
			mediaBucketName: "my-bucket",
			mediaFilepath:   "/audio/greeting.mp3",
		},
		{
			name: "creates_tts_with_empty_fields",

			gender:          "",
			text:            "",
			language:        "",
			mediaBucketName: "",
			mediaFilepath:   "",
		},
		{
			name: "creates_tts_with_female_gender",

			gender:          GenderFemale,
			text:            "Welcome to our service",
			language:        "en-GB",
			mediaBucketName: "audio-bucket",
			mediaFilepath:   "/sounds/welcome.wav",
		},
		{
			name: "creates_tts_with_neutral_gender",

			gender:          GenderNeutral,
			text:            "Your order is ready",
			language:        "ko-KR",
			mediaBucketName: "tts-bucket",
			mediaFilepath:   "/notifications/order.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tts := &TTS{
				Gender:          tt.gender,
				Text:            tt.text,
				Language:        tt.language,
				MediaBucketName: tt.mediaBucketName,
				MediaFilepath:   tt.mediaFilepath,
			}

			if tts.Gender != tt.gender {
				t.Errorf("Wrong Gender. expect: %s, got: %s", tt.gender, tts.Gender)
			}
			if tts.Text != tt.text {
				t.Errorf("Wrong Text. expect: %s, got: %s", tt.text, tts.Text)
			}
			if tts.Language != tt.language {
				t.Errorf("Wrong Language. expect: %s, got: %s", tt.language, tts.Language)
			}
			if tts.MediaBucketName != tt.mediaBucketName {
				t.Errorf("Wrong MediaBucketName. expect: %s, got: %s", tt.mediaBucketName, tts.MediaBucketName)
			}
			if tts.MediaFilepath != tt.mediaFilepath {
				t.Errorf("Wrong MediaFilepath. expect: %s, got: %s", tt.mediaFilepath, tts.MediaFilepath)
			}
		})
	}
}

func TestGenderConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Gender
		expected string
	}{
		{
			name:     "gender_male",
			constant: GenderMale,
			expected: "male",
		},
		{
			name:     "gender_female",
			constant: GenderFemale,
			expected: "female",
		},
		{
			name:     "gender_neutral",
			constant: GenderNeutral,
			expected: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
