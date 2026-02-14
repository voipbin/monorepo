package ttshandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/pkg/audiohandler"
	"monorepo/bin-tts-manager/pkg/buckethandler"

	"go.uber.org/mock/gomock"
)

func Test_isValidSSML(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		valid bool
	}{
		{
			name:  "valid ssml with speak tag",
			text:  "<speak>Hello world</speak>",
			valid: true,
		},
		{
			name:  "valid ssml with break tag",
			text:  "<speak>Hello<break time=\"500ms\"/>world</speak>",
			valid: true,
		},
		{
			name:  "valid ssml with emphasis tag",
			text:  "<speak><emphasis level=\"strong\">Hello</emphasis> world</speak>",
			valid: true,
		},
		{
			name:  "valid empty speak",
			text:  "<speak></speak>",
			valid: true,
		},
		{
			name:  "invalid ssml - unclosed tag",
			text:  "<speak>Hello",
			valid: false,
		},
		{
			name:  "invalid ssml - malformed",
			text:  "<speak><break",
			valid: false,
		},
		{
			name:  "plain text not valid",
			text:  "Hello world",
			valid: false,
		},
		{
			name:  "empty string not valid",
			text:  "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAudio := audiohandler.NewMockAudioHandler(mc)
			mockBucket := buckethandler.NewMockBucketHandler(mc)

			h := &ttsHandler{
				audioHandler:  mockAudio,
				bucketHandler: mockBucket,
			}

			result := h.isValidSSML(tt.text)
			if result != tt.valid {
				t.Errorf("expected %v, got %v for text: %q", tt.valid, result, tt.text)
			}
		})
	}
}

func Test_normalizeText_invalidSSML(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAudio := audiohandler.NewMockAudioHandler(mc)
	mockBucket := buckethandler.NewMockBucketHandler(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &ttsHandler{
		audioHandler:   mockAudio,
		bucketHandler:  mockBucket,
		requestHandler: mockRequest,
		notifyHandler:  mockNotify,
	}

	tests := []struct {
		name      string
		text      string
		expectErr bool
	}{
		{
			name:      "unclosed tag that can't be wrapped",
			text:      "<speak><break",
			expectErr: true,
		},
		{
			name:      "malformed XML",
			text:      "<><",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := h.normalizeText(ctx, tt.text)
			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
