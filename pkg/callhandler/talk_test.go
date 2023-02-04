package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	tmtts "gitlab.com/voipbin/bin-manager/tts-manager.git/models/tts"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_Talk(t *testing.T) {

	tests := []struct {
		name string

		callID   uuid.UUID
		runNext  bool
		text     string
		gender   string
		language string
		filename string

		responseCall *call.Call
		responseTTS  *tmtts.TTS

		expectURI      []string
		expectActionID uuid.UUID
	}{
		{
			"run next is true",

			uuid.FromStringOrNil("27bf2d84-a49c-11ed-aafc-e3364f827dc8"),
			true,
			`hello world`,
			"male",
			"en-US",
			"tts/tmp_filename.wav",

			&call.Call{
				ID:        uuid.FromStringOrNil("27bf2d84-a49c-11ed-aafc-e3364f827dc8"),
				ChannelID: "2830d52e-a49c-11ed-b9df-ef7dbeaeaf09",
				Action: fmaction.Action{
					ID:     uuid.FromStringOrNil("285a1c22-a49c-11ed-b48e-6f39f4fd59ff"),
					Type:   fmaction.TypeTalk,
					Option: []byte(`{"text":"hello world","gender":"male","language":"en-US"}`),
				},
			},

			&tmtts.TTS{
				Gender:          tmtts.GenderMale,
				Text:            "hello world",
				Language:        "en-US",
				MediaBucketName: "test_bucket",
				MediaFilepath:   "tts/tmp_filename.wav",
			},

			[]string{"sound:http://localhost:8000/temp/tts/tmp_filename.wav"},
			uuid.FromStringOrNil("285a1c22-a49c-11ed-b48e-6f39f4fd59ff"),
		},
		{
			"run next is false",

			uuid.FromStringOrNil("71e7f32c-a49d-11ed-8cc4-a300fe8c4c9d"),
			false,
			`hello world`,
			"male",
			"en-US",
			"tts/tmp_filename.wav",

			&call.Call{
				ID:        uuid.FromStringOrNil("71e7f32c-a49d-11ed-8cc4-a300fe8c4c9d"),
				ChannelID: "7218dd84-a49d-11ed-af2d-8f7a58f91a79",
				Action: fmaction.Action{
					ID:     uuid.FromStringOrNil("285a1c22-a49c-11ed-b48e-6f39f4fd59ff"),
					Type:   fmaction.TypeTalk,
					Option: []byte(`{"text":"hello world","gender":"male","language":"en-US"}`),
				},
			},

			&tmtts.TTS{
				Gender:          tmtts.GenderMale,
				Text:            "hello world",
				Language:        "en-US",
				MediaBucketName: "test_bucket",
				MediaFilepath:   "tts/tmp_filename.wav",
			},

			[]string{"sound:http://localhost:8000/temp/tts/tmp_filename.wav"},
			uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.callID).Return(tt.responseCall, nil)

			if tt.responseCall.Status != call.StatusProgressing {
				mockChannel.EXPECT().Answer(ctx, tt.responseCall.ChannelID).Return(nil)
			}

			mockReq.EXPECT().TTSV1SpeecheCreate(ctx, tt.responseCall.ID, tt.text, tmtts.Gender(tt.gender), tt.language, 10000).Return(tt.responseTTS, nil)
			mockChannel.EXPECT().Play(ctx, tt.responseCall.ChannelID, tt.expectActionID, tt.expectURI, "").Return(nil)

			if err := h.Talk(ctx, tt.callID, tt.runNext, tt.text, tt.gender, tt.language); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
