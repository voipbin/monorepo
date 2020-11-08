package ttshandler

import (
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/audiohandler"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/buckethandler"
)

func TestTTSCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAudio := audiohandler.NewMockAudioHandler(mc)
	mockBucket := buckethandler.NewMockBucketHandler(mc)

	h := &ttsHandler{
		audioHandler:  mockAudio,
		bucketHandler: mockBucket,
	}

	type test struct {
		name     string
		text     string
		gender   string
		language string
		filename string
	}

	tests := []test{
		{
			"normal",
			"<speak>Hello world</speak>",
			"male",
			"en-US",
			"865e9979dc81574475491a52a38bd423487935e9.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := fmt.Sprintf("%s/%s", bucketDirectory, tt.filename)

			mockBucket.EXPECT().FileExist(target).Return(false)
			mockAudio.EXPECT().AudioCreate(tt.text, tt.language, tt.gender, tt.filename).Return(nil)
			mockBucket.EXPECT().FileUpload(tt.filename, target).Return(nil)

			h.TTSCreate(tt.text, tt.language, tt.gender)
		})
	}
}
