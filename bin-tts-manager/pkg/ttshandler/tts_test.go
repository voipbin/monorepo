package ttshandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tts-manager/models/tts"
	"monorepo/bin-tts-manager/pkg/audiohandler"
	"monorepo/bin-tts-manager/pkg/buckethandler"
)

func Test_buildAttempts(t *testing.T) {
	tests := []struct {
		name     string
		provider tts.Provider
		voiceID  string
		expect   []providerAttempt
	}{
		{
			name:     "gcp provider",
			provider: tts.ProviderGCP,
			voiceID:  "en-US-Wavenet-F",
			expect: []providerAttempt{
				{provider: tts.ProviderGCP, voiceID: "en-US-Wavenet-F"},
				{provider: tts.ProviderAWS, voiceID: ""},
			},
		},
		{
			name:     "aws provider",
			provider: tts.ProviderAWS,
			voiceID:  "Joanna",
			expect: []providerAttempt{
				{provider: tts.ProviderAWS, voiceID: "Joanna"},
				{provider: tts.ProviderGCP, voiceID: ""},
			},
		},
		{
			name:     "empty provider defaults to gcp first",
			provider: "",
			voiceID:  "en-US-Wavenet-F",
			expect: []providerAttempt{
				{provider: tts.ProviderGCP, voiceID: "en-US-Wavenet-F"},
				{provider: tts.ProviderAWS, voiceID: ""},
			},
		},
		{
			name:     "empty provider with empty voice_id",
			provider: "",
			voiceID:  "",
			expect: []providerAttempt{
				{provider: tts.ProviderGCP, voiceID: ""},
				{provider: tts.ProviderAWS, voiceID: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAttempts(tt.provider, tt.voiceID)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, got)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	type test struct {
		name string

		callID   uuid.UUID
		text     string
		language string
		provider tts.Provider
		voiceID  string

		// normalizedText is what normalizeText returns. When empty, defaults to text (assumed already valid SSML).
		normalizedText string

		// primary attempt
		primaryCacheHit  bool
		primaryCreateErr error
		primaryFilePath  string
		primaryMediaPath string

		// fallback attempt (only used when primary fails and is not cache hit)
		fallbackCacheHit  bool
		fallbackCreateErr error
		fallbackFilePath  string
		fallbackMediaPath string

		expectRes *tts.TTS
		expectErr bool
	}

	tests := []test{
		{
			name: "primary succeeds (no fallback)",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			language: "en-US",
			provider: tts.ProviderGCP,
			voiceID:  "en-US-Wavenet-F",

			primaryFilePath:  "/shared-data/d27dff3751181a10e4fc7f216a936c9c3f33c339.wav",
			primaryMediaPath: "http://10-96-0-112.bin-manager.pod.cluster.local/d27dff3751181a10e4fc7f216a936c9c3f33c339.wav",

			expectRes: &tts.TTS{
				Provider:      tts.ProviderGCP,
				VoiceID:       "en-US-Wavenet-F",
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://10-96-0-112.bin-manager.pod.cluster.local/d27dff3751181a10e4fc7f216a936c9c3f33c339.wav",
			},
		},
		{
			name: "primary cache hit skips audio creation",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			language: "en-US",
			provider: tts.ProviderGCP,
			voiceID:  "en-US-Wavenet-F",

			primaryCacheHit:  true,
			primaryFilePath:  "/shared-data/d27dff3751181a10e4fc7f216a936c9c3f33c339.wav",
			primaryMediaPath: "http://10-96-0-112.bin-manager.pod.cluster.local/d27dff3751181a10e4fc7f216a936c9c3f33c339.wav",

			expectRes: &tts.TTS{
				Provider:      tts.ProviderGCP,
				VoiceID:       "en-US-Wavenet-F",
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://10-96-0-112.bin-manager.pod.cluster.local/d27dff3751181a10e4fc7f216a936c9c3f33c339.wav",
			},
		},
		{
			name: "primary fails fallback cache hit",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			language: "en-US",
			provider: tts.ProviderGCP,
			voiceID:  "en-US-Wavenet-F",

			primaryCreateErr: fmt.Errorf("gcp error"),
			primaryFilePath:  "/shared-data/primary.wav",
			primaryMediaPath: "http://host/primary.wav",

			fallbackCacheHit:  true,
			fallbackFilePath:  "/shared-data/fallback.wav",
			fallbackMediaPath: "http://host/fallback.wav",

			expectRes: &tts.TTS{
				Provider:      tts.ProviderAWS,
				VoiceID:       "",
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://host/fallback.wav",
			},
		},
		{
			name: "primary fails fallback audio creation succeeds",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			language: "en-US",
			provider: tts.ProviderGCP,
			voiceID:  "en-US-Wavenet-F",

			primaryCreateErr: fmt.Errorf("gcp error"),
			primaryFilePath:  "/shared-data/primary.wav",
			primaryMediaPath: "http://host/primary.wav",

			fallbackFilePath:  "/shared-data/fallback.wav",
			fallbackMediaPath: "http://host/fallback.wav",

			expectRes: &tts.TTS{
				Provider:      tts.ProviderAWS,
				VoiceID:       "",
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://host/fallback.wav",
			},
		},
		{
			name: "both providers fail",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			language: "en-US",
			provider: tts.ProviderGCP,
			voiceID:  "en-US-Wavenet-F",

			primaryCreateErr: fmt.Errorf("gcp error"),
			primaryFilePath:  "/shared-data/primary.wav",
			primaryMediaPath: "http://host/primary.wav",

			fallbackCreateErr: fmt.Errorf("aws error"),
			fallbackFilePath:  "/shared-data/fallback.wav",
			fallbackMediaPath: "http://host/fallback.wav",

			expectErr: true,
		},
		{
			name: "empty provider resolves to gcp first then aws",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			language: "en-US",
			provider: "",
			voiceID:  "en-US-Wavenet-F",

			primaryCreateErr: fmt.Errorf("gcp error"),
			primaryFilePath:  "/shared-data/primary.wav",
			primaryMediaPath: "http://host/primary.wav",

			fallbackFilePath:  "/shared-data/fallback.wav",
			fallbackMediaPath: "http://host/fallback.wav",

			expectRes: &tts.TTS{
				Provider:      tts.ProviderAWS,
				VoiceID:       "",
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://host/fallback.wav",
			},
		},
		{
			name: "aws provider falls back to gcp",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			language: "en-US",
			provider: tts.ProviderAWS,
			voiceID:  "Joanna",

			primaryCreateErr: fmt.Errorf("aws error"),
			primaryFilePath:  "/shared-data/primary.wav",
			primaryMediaPath: "http://host/primary.wav",

			fallbackFilePath:  "/shared-data/fallback.wav",
			fallbackMediaPath: "http://host/fallback.wav",

			expectRes: &tts.TTS{
				Provider:      tts.ProviderGCP,
				VoiceID:       "",
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://host/fallback.wav",
			},
		},
		{
			name: "plain text is normalized before cache key computation",

			callID:         uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:           "Hello world",
			normalizedText: "<speak>Hello world</speak>",
			language:       "en-US",
			provider:       tts.ProviderGCP,
			voiceID:        "en-US-Wavenet-F",

			primaryFilePath:  "/shared-data/normalized.wav",
			primaryMediaPath: "http://host/normalized.wav",

			expectRes: &tts.TTS{
				Provider:      tts.ProviderGCP,
				VoiceID:       "en-US-Wavenet-F",
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://host/normalized.wav",
			},
		},
		{
			name: "empty provider and empty voice_id both attempts use defaults",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			language: "en-US",
			provider: "",
			voiceID:  "",

			primaryCreateErr: fmt.Errorf("gcp error"),
			primaryFilePath:  "/shared-data/primary.wav",
			primaryMediaPath: "http://host/primary.wav",

			fallbackFilePath:  "/shared-data/fallback.wav",
			fallbackMediaPath: "http://host/fallback.wav",

			expectRes: &tts.TTS{
				Provider:      tts.ProviderAWS,
				VoiceID:       "",
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://host/fallback.wav",
			},
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
			ctx := context.Background()

			// resolve normalized text: if not specified, text is already valid SSML
			normalizedText := tt.normalizedText
			if normalizedText == "" {
				normalizedText = tt.text
			}

			// determine the attempts that will be made
			attempts := buildAttempts(tt.provider, tt.voiceID)

			// set up expectations for primary attempt
			primary := attempts[0]
			primaryFilename := h.filenameHashGenerator(normalizedText, tt.language, primary.provider, primary.voiceID)

			mockBucket.EXPECT().OSGetFilepath(ctx, primaryFilename).Return(tt.primaryFilePath)
			mockBucket.EXPECT().OSGetMediaFilepath(ctx, primaryFilename).Return(tt.primaryMediaPath)
			mockBucket.EXPECT().OSFileExist(ctx, tt.primaryFilePath).Return(tt.primaryCacheHit)

			if !tt.primaryCacheHit {
				mockAudio.EXPECT().AudioCreate(ctx, tt.callID, normalizedText, tt.language, primary.provider, primary.voiceID, tt.primaryFilePath).Return(tt.primaryCreateErr)

				// if primary failed, set up fallback expectations
				if tt.primaryCreateErr != nil && len(attempts) > 1 {
					fallback := attempts[1]
					fallbackFilename := h.filenameHashGenerator(normalizedText, tt.language, fallback.provider, fallback.voiceID)

					mockBucket.EXPECT().OSGetFilepath(ctx, fallbackFilename).Return(tt.fallbackFilePath)
					mockBucket.EXPECT().OSGetMediaFilepath(ctx, fallbackFilename).Return(tt.fallbackMediaPath)
					mockBucket.EXPECT().OSFileExist(ctx, tt.fallbackFilePath).Return(tt.fallbackCacheHit)

					if !tt.fallbackCacheHit {
						mockAudio.EXPECT().AudioCreate(ctx, tt.callID, normalizedText, tt.language, fallback.provider, fallback.voiceID, tt.fallbackFilePath).Return(tt.fallbackCreateErr)
					}
				}
			}

			res, err := h.Create(ctx, tt.callID, tt.text, tt.language, tt.provider, tt.voiceID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Create_normalizationFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAudio := audiohandler.NewMockAudioHandler(mc)
	mockBucket := buckethandler.NewMockBucketHandler(mc)

	h := &ttsHandler{
		audioHandler:  mockAudio,
		bucketHandler: mockBucket,
	}
	ctx := context.Background()

	// invalid XML that can't be normalized: unclosed tag without valid wrapping
	invalidText := "<speak><break"

	res, err := h.Create(ctx, uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"), invalidText, "en-US", tts.ProviderGCP, "en-US-Wavenet-F")
	if err == nil {
		t.Errorf("Expected normalization error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result on normalization failure, got: %v", res)
	}
}

func Test_filenameHashGenerator(t *testing.T) {

	type test struct {
		name string

		text     string
		language string
		provider tts.Provider
		voiceID  string

		expectRes string
	}

	tests := []test{
		{
			name: "normal",

			text:     "Hello, welcome to the voipbin! This is test message. Please feel free to enjoy the voipbin service.",
			language: "en-US",
			provider: tts.ProviderGCP,
			voiceID:  "en-US-Wavenet-F",

			expectRes: "24195475b3e0012c9237b1a525abbcda22f89d6c.wav",
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

			res := h.filenameHashGenerator(tt.text, tt.language, tt.provider, tt.voiceID)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_filenameHashGenerator_differentInputs(t *testing.T) {
	h := &ttsHandler{}

	baseHash := h.filenameHashGenerator("hello", "en-US", tts.ProviderGCP, "en-US-Wavenet-F")

	tests := []struct {
		name     string
		text     string
		language string
		provider tts.Provider
		voiceID  string
	}{
		{
			name:     "different provider produces different hash",
			text:     "hello",
			language: "en-US",
			provider: tts.ProviderAWS,
			voiceID:  "en-US-Wavenet-F",
		},
		{
			name:     "different voice_id produces different hash",
			text:     "hello",
			language: "en-US",
			provider: tts.ProviderGCP,
			voiceID:  "en-US-Wavenet-A",
		},
		{
			name:     "empty provider produces different hash",
			text:     "hello",
			language: "en-US",
			provider: "",
			voiceID:  "en-US-Wavenet-F",
		},
		{
			name:     "empty voice_id produces different hash",
			text:     "hello",
			language: "en-US",
			provider: tts.ProviderGCP,
			voiceID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := h.filenameHashGenerator(tt.text, tt.language, tt.provider, tt.voiceID)
			if hash == baseHash {
				t.Errorf("expected different hash, got same: %s", hash)
			}
		})
	}
}

func Test_normalizeSSML(t *testing.T) {

	type test struct {
		name string

		text      string
		expectRes string
	}

	tests := []test{
		{
			name: "have no <speak>",

			text:      "Hello, welcome to the voipbin!",
			expectRes: "<speak>Hello, welcome to the voipbin!</speak>",
		},
		{
			name: "have <speak>",

			text:      "<speak>Hello, welcome to the voipbin!</speak>",
			expectRes: "<speak>Hello, welcome to the voipbin!</speak>",
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
			ctx := context.Background()

			res, err := h.normalizeText(ctx, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}
