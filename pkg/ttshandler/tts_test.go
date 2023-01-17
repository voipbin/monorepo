package ttshandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/models/tts"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/audiohandler"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/buckethandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		callID   uuid.UUID
		text     string
		gender   tts.Gender
		language string
		filename string

		expectRes *tts.TTS
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("c1a8bfe6-9214-11ec-a013-1bbdbd87fc23"),
			"<speak>Hello world</speak>",
			tts.GenderFemale,
			"en-US",
			"766e587168455d862b8ef2a931341e7adaa106e1.wav",

			&tts.TTS{
				Gender:        tts.GenderFemale,
				Text:          "<speak>Hello world</speak>",
				Language:      "en-US",
				MediaFilepath: "temp/tts/766e587168455d862b8ef2a931341e7adaa106e1.wav",
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

			target := fmt.Sprintf("%s/%s", bucketDirectory, tt.filename)

			mockBucket.EXPECT().FileExist(ctx, target).Return(false)
			mockAudio.EXPECT().AudioCreate(ctx, tt.callID, tt.text, tt.language, tt.gender, tt.filename).Return(nil)
			mockBucket.EXPECT().FileUpload(ctx, tt.filename, target).Return(nil)

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
