package callhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	tmtts "monorepo/bin-tts-manager/models/tts"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/playback"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Talk(t *testing.T) {

	tests := []struct {
		name string

		callID   uuid.UUID
		runNext  bool
		text     string
		gender   string
		language string

		responseCall         *call.Call
		responseTTS          *tmtts.TTS
		responseUUIDActionID uuid.UUID

		expectURI        []string
		expectPlaybackID string
	}{
		{
			name: "run next is true",

			callID:   uuid.FromStringOrNil("27bf2d84-a49c-11ed-aafc-e3364f827dc8"),
			runNext:  true,
			text:     `hello world`,
			gender:   "male",
			language: "en-US",

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27bf2d84-a49c-11ed-aafc-e3364f827dc8"),
				},
				ChannelID: "2830d52e-a49c-11ed-b9df-ef7dbeaeaf09",
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("285a1c22-a49c-11ed-b48e-6f39f4fd59ff"),
					Type: fmaction.TypeTalk,
					Option: map[string]any{
						"text":     "hello world",
						"gender":   "male",
						"language": "en-US",
					},
				},
			},
			responseTTS: &tmtts.TTS{
				Gender:          tmtts.GenderMale,
				Text:            "hello world",
				Language:        "en-US",
				MediaBucketName: "test_bucket",
				MediaFilepath:   "http://10-96-0-112.bin-manager.pod.cluster.local/tmp_filename.wav",
			},

			expectURI:        []string{"sound:http://10-96-0-112.bin-manager.pod.cluster.local/tmp_filename.wav"},
			expectPlaybackID: playback.IDPrefixCall + "285a1c22-a49c-11ed-b48e-6f39f4fd59ff",
		},
		{
			name: "run next is false",

			callID:   uuid.FromStringOrNil("71e7f32c-a49d-11ed-8cc4-a300fe8c4c9d"),
			runNext:  false,
			text:     `hello world`,
			gender:   "male",
			language: "en-US",

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("71e7f32c-a49d-11ed-8cc4-a300fe8c4c9d"),
				},
				ChannelID: "7218dd84-a49d-11ed-af2d-8f7a58f91a79",
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("285a1c22-a49c-11ed-b48e-6f39f4fd59ff"),
					Type: fmaction.TypeTalk,
					Option: map[string]any{
						"text":     "hello world",
						"gender":   "male",
						"language": "en-US",
					},
				},
			},
			responseTTS: &tmtts.TTS{
				Gender:          tmtts.GenderMale,
				Text:            "hello world",
				Language:        "en-US",
				MediaBucketName: "test_bucket",
				MediaFilepath:   "http://10-96-0-112.bin-manager.pod.cluster.local/tmp_filename.wav",
			},
			responseUUIDActionID: uuid.FromStringOrNil("c988dc86-7de9-11f0-b57e-57ffac78c55a"),

			expectURI:        []string{"sound:http://10-96-0-112.bin-manager.pod.cluster.local/tmp_filename.wav"},
			expectPlaybackID: playback.IDPrefixCall + "c988dc86-7de9-11f0-b57e-57ffac78c55a",
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

			mockDB.EXPECT().CallGet(ctx, tt.callID.Return(tt.responseCall, nil)

			if tt.responseCall.Status != call.StatusProgressing {
				mockChannel.EXPECT().Answer(ctx, tt.responseCall.ChannelID.Return(nil)
			}

			mockReq.EXPECT().TTSV1SpeecheCreate(ctx, tt.responseCall.ID, tt.text, tmtts.Gender(tt.gender), tt.language, 10000.Return(tt.responseTTS, nil)
			if !tt.runNext {
				mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDActionID)
			}
			mockChannel.EXPECT().Play(ctx, tt.responseCall.ChannelID, tt.expectPlaybackID, tt.expectURI, "", 0, 0.Return(nil)

			if err := h.Talk(ctx, tt.callID, tt.runNext, tt.text, tt.gender, tt.language); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Play(t *testing.T) {

	tests := []struct {
		name string

		callID  uuid.UUID
		runNext bool
		urls    []string

		responseCall *call.Call

		expectURI        []string
		expectplaybackID string
	}{
		{
			name: "run next is true",

			callID:  uuid.FromStringOrNil("9a78ee81-8462-4350-9fd2-6bbf93cc26f2"),
			runNext: true,
			urls: []string{
				"https://test.com/test-1.wav",
				"https://test.com/test-2.wav",
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9a78ee81-8462-4350-9fd2-6bbf93cc26f2"),
				},
				ChannelID: "c486e658-25b2-416b-a5df-0d11c7c633ab",
				Action: fmaction.Action{
					ID: uuid.FromStringOrNil("19baefc2-5de9-4acb-b6a7-0f4a55089934"),
				},
			},

			expectURI: []string{
				"sound:https://test.com/test-1.wav",
				"sound:https://test.com/test-2.wav",
			},
			expectplaybackID: playback.IDPrefixCall + "19baefc2-5de9-4acb-b6a7-0f4a55089934",
		},
		{
			name: "run next is false",

			callID:  uuid.FromStringOrNil("3d0c6c4a-a17f-4c5d-ae2b-635e5866fbd0"),
			runNext: false,
			urls: []string{
				"https://test.com/139ddf48-1110-4ba1-b3b3-be4ded864089.wav",
				"https://test.com/0d604e92-96cc-4521-a787-9745f0ae70c3.wav",
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d0c6c4a-a17f-4c5d-ae2b-635e5866fbd0"),
				},
				ChannelID: "d13fff6a-99ab-4a04-b5d3-754ab00efb00",
				Action: fmaction.Action{
					ID: uuid.FromStringOrNil("abe00434-bcfb-445d-a8b1-33d936f3ebc3"),
				},
			},

			expectURI: []string{
				"sound:https://test.com/139ddf48-1110-4ba1-b3b3-be4ded864089.wav",
				"sound:https://test.com/0d604e92-96cc-4521-a787-9745f0ae70c3.wav",
			},
			expectplaybackID: playback.IDPrefixCall + "00000000-0000-0000-0000-000000000000",
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

			mockDB.EXPECT().CallGet(ctx, tt.callID.Return(tt.responseCall, nil)
			mockChannel.EXPECT().Play(ctx, tt.responseCall.ChannelID, tt.expectplaybackID, tt.expectURI, "", 0, 0.Return(nil)

			if err := h.Play(ctx, tt.callID, tt.runNext, tt.urls); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_MediaStop(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		responseCall *call.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("40c63616-5a28-43f8-b016-5f198f155535"),

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("40c63616-5a28-43f8-b016-5f198f155535"),
				},
				ChannelID: "996e51e1-0d70-4115-89e8-20f9fc1ea45c",
			},
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

			mockDB.EXPECT().CallGet(ctx, tt.callID.Return(tt.responseCall, nil)
			mockChannel.EXPECT().PlaybackStop(ctx, tt.responseCall.ChannelID.Return(nil)

			if err := h.MediaStop(ctx, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
