package tts

import (
	"testing"
)

func TestTTS(t *testing.T) {
	tests := []struct {
		name string

		provider        Provider
		voiceID         string
		text            string
		language        string
		mediaBucketName string
		mediaFilepath   string
	}{
		{
			name: "creates_tts_with_all_fields",

			provider:        ProviderGCP,
			voiceID:         "en-US-Wavenet-F",
			text:            "Hello, world!",
			language:        "en-US",
			mediaBucketName: "my-bucket",
			mediaFilepath:   "/audio/greeting.mp3",
		},
		{
			name: "creates_tts_with_empty_fields",

			provider:        "",
			voiceID:         "",
			text:            "",
			language:        "",
			mediaBucketName: "",
			mediaFilepath:   "",
		},
		{
			name: "creates_tts_with_aws_provider",

			provider:        ProviderAWS,
			voiceID:         "Joanna",
			text:            "Welcome to our service",
			language:        "en-GB",
			mediaBucketName: "audio-bucket",
			mediaFilepath:   "/sounds/welcome.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tts := &TTS{
				Provider:        tt.provider,
				VoiceID:         tt.voiceID,
				Text:            tt.text,
				Language:        tt.language,
				MediaBucketName: tt.mediaBucketName,
				MediaFilepath:   tt.mediaFilepath,
			}

			if tts.Provider != tt.provider {
				t.Errorf("Wrong Provider. expect: %s, got: %s", tt.provider, tts.Provider)
			}
			if tts.VoiceID != tt.voiceID {
				t.Errorf("Wrong VoiceID. expect: %s, got: %s", tt.voiceID, tts.VoiceID)
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

func TestProviderConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Provider
		expected string
	}{
		{
			name:     "provider_gcp",
			constant: ProviderGCP,
			expected: "gcp",
		},
		{
			name:     "provider_aws",
			constant: ProviderAWS,
			expected: "aws",
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
