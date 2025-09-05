package ttshandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tts-manager/models/tts"
	"monorepo/bin-tts-manager/pkg/audiohandler"
	"monorepo/bin-tts-manager/pkg/buckethandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		callID   uuid.UUID
		text     string
		gender   tts.Gender
		language string

		responseFilePath      string
		responseMediaFilepath string

		expectFilename string
		expectRes      *tts.TTS
	}

	tests := []test{
		{
			name: "normal",

			callID:   uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			text:     "<speak>Hello world</speak>",
			gender:   tts.GenderFemale,
			language: "en-US",

			responseFilePath:      "/tmp/766e587168455d862b8ef2a931341e7adaa106e1.wav",
			responseMediaFilepath: "http://10-96-0-112.bin-manager.pod.cluster.local/766e587168455d862b8ef2a931341e7adaa106e1.wav",

			expectFilename: "766e587168455d862b8ef2a931341e7adaa106e1.wav",
			expectRes: &tts.TTS{
				Gender:        tts.GenderFemale,
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "http://10-96-0-112.bin-manager.pod.cluster.local/766e587168455d862b8ef2a931341e7adaa106e1.wav",
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

			mockBucket.EXPECT().OSGetFilepath(ctx, tt.expectFilename).Return(tt.responseFilePath)
			mockBucket.EXPECT().OSGetMediaFilepath(ctx, tt.expectFilename).Return(tt.responseMediaFilepath)
			mockBucket.EXPECT().OSFileExist(ctx, tt.responseFilePath).Return(false)
			mockAudio.EXPECT().AudioCreate(ctx, tt.callID, tt.text, tt.language, tt.gender, tt.responseFilePath).Return(nil)

			res, err := h.Create(ctx, tt.callID, tt.text, tt.language, tt.gender)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_filenameHashGenerator(t *testing.T) {

	type test struct {
		name string

		text     string
		gender   tts.Gender
		language string

		expectRes string
	}

	tests := []test{
		{
			name: "normal",

			text:     "Hello, welcome to the voipbin! This is test message. Please feel free to enjoy the voipbin service.",
			gender:   tts.GenderFemale,
			language: "en-US",

			expectRes: "1e8561db13fe0f60473e9708cad94c339c018328.wav",
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

			res := h.filenameHashGenerator(tt.text, tt.language, tt.gender)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
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
