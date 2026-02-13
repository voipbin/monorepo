package callhandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/variable"

	tmtts "monorepo/bin-tts-manager/models/tts"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	callapplication "monorepo/bin-call-manager/models/callapplication"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/playback"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

func Test_ActionExecute_actionExecuteConfbridgeJoin(t *testing.T) {

	tests := []struct {
		name               string
		call               *call.Call
		expectConfbridgeID uuid.UUID
	}{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ed1620aa-3e6e-11ec-902b-170b2849173a"),
				},
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("012c12c8-7552-4ec9-85b7-d39d78fa789f"),
					Type: fmaction.TypeConfbridgeJoin,
					Option: map[string]any{
						"confbridge_id": "3f5ff42c-3e6e-11ec-8c17-039eb294368c",
					},
				},
			},
			uuid.FromStringOrNil("3f5ff42c-3e6e-11ec-8c17-039eb294368c"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &callHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				confbridgeHandler: mockConfbridge,
			}

			ctx := context.Background()

			mockConfbridge.EXPECT().Join(ctx, tt.expectConfbridgeID, tt.call.ID)
			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteStreamEcho(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call
	}{
		{
			"empty option",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("741a0f32-6bb2-11ed-bf34-9b3f75da5e87"),
				},
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("d5ab7c42-10dc-4363-8f06-89443f9a2c3b"),
					Type: fmaction.TypeStreamEcho,
				},
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

			mockChannel.EXPECT().Continue(ctx, tt.call.ChannelID, "svc-stream_echo", "s", 1, "").Return(nil)
			mockReq.EXPECT().CallV1CallActionTimeout(ctx, tt.call.ID, gomock.Any(), &tt.call.Action).Return(nil)
			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteAnswer(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call
	}{
		{
			"empty option",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4371b0d6-df48-11ea-9a8c-177968c165e9"),
				},
				ChannelID: "5b21353a-df48-11ea-8207-6fc0fa36a3fe",
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("87ea1d71-dd34-4c53-bd82-14082d5944aa"),
					Type: fmaction.TypeAnswer,
				},
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

			mockChannel.EXPECT().Answer(ctx, tt.call.ChannelID).Return(nil)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)
			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionTimeoutNext(t *testing.T) {

	tests := []struct {
		name            string
		call            *call.Call
		action          *fmaction.Action
		responseChannel *channel.Channel
	}{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("66e039c6-e3fc-11ea-ae6f-53584373e7c9"),
				},
				ChannelID: "12a05228-e3fd-11ea-b55f-afd68e7aa755",
				Action: fmaction.Action{
					ID:        uuid.FromStringOrNil("b44bae7a-e3fc-11ea-a908-374a03455628"),
					TMExecute: func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }(),
				},
			},
			&fmaction.Action{
				ID:        uuid.FromStringOrNil("b44bae7a-e3fc-11ea-a908-374a03455628"),
				Type:      fmaction.TypeAnswer,
				TMExecute: func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }(),
			},
			&channel.Channel{
				ID: "12a05228-e3fd-11ea-b55f-afd68e7aa755",
				Data: map[string]interface{}{
					"CONTEXT": "conf-in",
				},
				StasisName: "call-in",
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

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockChannel.EXPECT().Get(ctx, tt.call.ChannelID).Return(tt.responseChannel, nil)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)

			if err := h.ActionTimeout(ctx, tt.call.ID, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteTalk(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseTTS *tmtts.TTS

		expectSSML       string
		expectLanguage   string
		expectProvider   string
		expectVoiceID    string
		expectPlaybackID string
		expectURI        []string
		expectAsync      bool
	}{
		{
			name: "normal",

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5e9f3946-2188-11eb-9d74-bf4bf1239da3"),
				},
				ChannelID: "61a1345a-2188-11eb-ba52-af82c1239d8f",
				Action: fmaction.Action{
					Type: fmaction.TypeTalk,
					ID:   uuid.FromStringOrNil("5c9cd6be-2195-11eb-a9c9-bfc91ac88411"),
					Option: map[string]any{
						"text":     "hello world",
						"language": "en-US",
					},
				},
			},

			responseTTS: &tmtts.TTS{
				Text:          "hello world",
				Language:      "en-US",
				MediaFilepath: "http://10-96-0-112.bin-manager.pod.cluster.local/tmp_filename.wav",
			},

			expectSSML:       `hello world`,
			expectLanguage:   "en-US",
			expectProvider:   "",
			expectVoiceID:    "",
			expectPlaybackID: playback.IDPrefixCall + "5c9cd6be-2195-11eb-a9c9-bfc91ac88411",
			expectURI:        []string{"sound:http://10-96-0-112.bin-manager.pod.cluster.local/tmp_filename.wav"},
		},
		{
			name: "async talk",

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cd53988e-cb3b-11f0-bba8-a36a06c0abb1"),
				},
				ChannelID: "cdb2ac20-cb3b-11f0-977b-03bf6ac80420",
				Action: fmaction.Action{
					Type: fmaction.TypeTalk,
					ID:   uuid.FromStringOrNil("cddc2ea6-cb3b-11f0-ac41-d340351b406c"),
					Option: map[string]any{
						"text":     "hello world",
						"language": "en-US",
						"async":    true,
					},
				},
			},

			responseTTS: &tmtts.TTS{
				Text:          "hello world",
				Language:      "en-US",
				MediaFilepath: "http://10-96-0-112.bin-manager.pod.cluster.local/tmp_filename.wav",
			},

			expectSSML:       `hello world`,
			expectLanguage:   "en-US",
			expectProvider:   "",
			expectVoiceID:    "",
			expectPlaybackID: playback.IDPrefixCall + "cddc2ea6-cb3b-11f0-ac41-d340351b406c",
			expectURI:        []string{"sound:http://10-96-0-112.bin-manager.pod.cluster.local/tmp_filename.wav"},
			expectAsync:      true,
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

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			if tt.call.Status != call.StatusProgressing {
				mockChannel.EXPECT().Answer(ctx, tt.call.ChannelID).Return(nil)
			}
			mockReq.EXPECT().TTSV1SpeecheCreate(ctx, tt.call.ID, tt.expectSSML, tt.expectLanguage, tmtts.Provider(tt.expectProvider), tt.expectVoiceID, 10000).Return(tt.responseTTS, nil)
			mockChannel.EXPECT().Play(ctx, tt.call.ChannelID, tt.expectPlaybackID, tt.expectURI, "", 0, 0).Return(nil)

			if tt.expectAsync {
				mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)
			}

			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecutePlay(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		expectPlaybackID string
		expectMedias     []string
	}{
		{
			name: "normal",

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f4fce23b-356f-45e2-994a-2e7763fb1f6a"),
				},
				ChannelID: "23ae2630-70c0-4a6d-a41f-aea675c24e75",
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("2641f116-90d1-40a3-a817-e16b4f3325da"),
					Type: fmaction.TypePlay,
					Option: map[string]any{
						"stream_urls": []string{
							"https://test.com/e258dc64-edb1-4220-8245-16ddc4941f96.wav",
							"https://test.com/84e3e1eb-4977-4ba3-bdf0-c56f35e24d57.wav",
						},
					},
				},
			},

			expectPlaybackID: playback.IDPrefixCall + "2641f116-90d1-40a3-a817-e16b4f3325da",
			expectMedias: []string{
				"sound:https://test.com/e258dc64-edb1-4220-8245-16ddc4941f96.wav",
				"sound:https://test.com/84e3e1eb-4977-4ba3-bdf0-c56f35e24d57.wav",
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

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockChannel.EXPECT().Play(ctx, tt.call.ChannelID, tt.expectPlaybackID, tt.expectMedias, "", 0, 0).Return(nil)

			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteRecordingStart(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		responseRecording *recording.Recording

		expectFormat       recording.Format
		expectEndOfSilence int
		expectEndOfKey     string
		expectDuration     int
		expectOnEndFlowID  uuid.UUID
	}{
		{
			name: "default",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bf4ff828-2a77-11eb-a984-33588027b8c4"),
				},
				ActiveflowID: uuid.FromStringOrNil("6853663a-0728-11f0-857c-ab4a6094268b"),
				ChannelID:    "bfd0e668-2a77-11eb-9993-e72b323b1801",
				Status:       call.StatusProgressing,
				Action: fmaction.Action{
					Type: fmaction.TypeRecordingStart,
					ID:   uuid.FromStringOrNil("c06f25c6-2a77-11eb-bcc8-e3d864a76f78"),
					Option: map[string]any{
						"format":         "wav",
						"end_of_silence": 5,
						"end_of_key":     "1",
						"duration":       600,
						"on_end_flow_id": "8af57a9a-0546-11f0-beb7-731fcb5a9acf",
					},
				},
			},

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dec99d2a-8fe8-11ed-b223-478f994dc5a0"),
				},
			},

			expectFormat:       recording.FormatWAV,
			expectEndOfSilence: 5,
			expectEndOfKey:     "1",
			expectDuration:     600,
			expectOnEndFlowID:  uuid.FromStringOrNil("8af57a9a-0546-11f0-beb7-731fcb5a9acf"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				recordingHandler: mockRecording,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockRecording.EXPECT().Start(
				ctx,
				tt.call.ActiveflowID,
				recording.ReferenceTypeCall,
				tt.call.ID,
				tt.expectFormat,
				tt.expectEndOfSilence,
				tt.expectEndOfKey,
				tt.expectDuration,
				tt.expectOnEndFlowID,
			).Return(tt.responseRecording, nil)
			mockDB.EXPECT().CallSetRecordingID(ctx, tt.call.ID, tt.responseRecording.ID).Return(nil)
			mockDB.EXPECT().CallAddRecordingIDs(ctx, tt.call.ID, tt.responseRecording.ID).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)
			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteRecordingStop(t *testing.T) {

	tests := []struct {
		name              string
		call              *call.Call
		responseRecording *recording.Recording
	}{
		{
			"default",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4dde92d0-2b9e-11eb-ad28-f732fd0afed7"),
				},
				ChannelID:   "5293419a-2b9e-11eb-bfa6-97a4312177f2",
				RecordingID: uuid.FromStringOrNil("b230d160-611f-11eb-9bee-2734cae1cab5"),
				Status:      call.StatusProgressing,
				Action: fmaction.Action{
					Type: fmaction.TypeRecordingStop,
					ID:   uuid.FromStringOrNil("4a3925dc-2b9e-11eb-abb3-d759c4b283d0"),
				},
			},
			&recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b230d160-611f-11eb-9bee-2734cae1cab5"),
				},
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
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				recordingHandler: mockRecording,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockRecording.EXPECT().Stop(ctx, tt.call.RecordingID).Return(tt.responseRecording, nil)
			mockDB.EXPECT().CallSetRecordingID(ctx, tt.call.ID, uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)

			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteDigitsReceive(t *testing.T) {

	tests := []struct {
		name     string
		call     *call.Call
		duration int

		responseVariable *variable.Variable
	}{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				},
				ChannelID: "c34e2226-6959-11eb-b57a-8718398e2ffc",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					ID:   uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
					Option: map[string]any{
						"duration": 1000,
						"length":   3,
					},
				},
			},
			1000,
			&variable.Variable{
				Variables: map[string]string{},
			},
		},
		{
			"finish on key set but not qualified",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				},
				ChannelID: "c34e2226-6959-11eb-b57a-8718398e2ffc",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					ID:   uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
					Option: map[string]any{
						"duration": 1000,
						"length":   3,
						"key":      "1234567",
					},
				},
			},
			1000,

			&variable.Variable{
				Variables: map[string]string{},
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

			h := &callHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				db:          mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.call.ActiveflowID).Return(tt.responseVariable, nil)
			mockReq.EXPECT().CallV1CallActionTimeout(ctx, tt.call.ID, tt.duration, &tt.call.Action).Return(nil)

			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteDTMFReceiveFinishWithStoredDTMFs(t *testing.T) {

	tests := []struct {
		name string

		responseCall     *call.Call
		responseVariable *variable.Variable
	}{
		{
			"length qualified",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				},
				ActiveflowID: uuid.FromStringOrNil("8ab35caa-df01-11ec-a567-abb76662ef08"),
				ChannelID:    "c34e2226-6959-11eb-b57a-8718398e2ffc",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					ID:   uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
					Option: map[string]any{
						"duration": 1000,
						"length":   3,
					},
				},
			},
			&variable.Variable{
				ID: uuid.FromStringOrNil("8ab35caa-df01-11ec-a567-abb76662ef08"),
				Variables: map[string]string{
					variableCallDigits: "123",
				},
			},
		},
		{
			"finish on key #",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				},
				ActiveflowID: uuid.FromStringOrNil("bc06ef06-df01-11ec-ad88-074454252454"),
				ChannelID:    "c34e2226-6959-11eb-b57a-8718398e2ffc",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					ID:   uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
					Option: map[string]any{
						"duration": 1000,
						"length":   3,
						"key":      "#",
					},
				},
			},
			&variable.Variable{
				ID: uuid.FromStringOrNil("bc06ef06-df01-11ec-ad88-074454252454"),
				Variables: map[string]string{
					variableCallDigits: "#",
				},
			},
		},
		{
			"finish on key *",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				},
				ActiveflowID: uuid.FromStringOrNil("e28f7a44-df01-11ec-8eaf-47af6e21909e"),
				ChannelID:    "c34e2226-6959-11eb-b57a-8718398e2ffc",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsReceive,
					ID:   uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
					Option: map[string]any{
						"duration": 1000,
						"length":   3,
						"key":      "*",
					},
				},
			},
			&variable.Variable{
				ID: uuid.FromStringOrNil("e28f7a44-df01-11ec-8eaf-47af6e21909e"),
				Variables: map[string]string{
					variableCallDigits: "*",
				},
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

			h := &callHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				db:          mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.responseCall.ActiveflowID).Return(tt.responseVariable, nil)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.responseCall.ID, false).Return(nil)

			if err := h.actionExecute(ctx, tt.responseCall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteDigitsSend(t *testing.T) {

	tests := []struct {
		name           string
		call           *call.Call
		expectDigits   string
		expectDuration int
		expectInterval int
		expectTimeout  int
	}{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("50270fae-69bf-11eb-a0a7-273260ea280c"),
				},
				ChannelID: "5daefc0e-69bf-11eb-9e3a-b7d9a5988373",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsSend,
					ID:   uuid.FromStringOrNil("508063d8-69bf-11eb-a668-abdbd47ce266"),
					Option: map[string]any{
						"digits":   "12345",
						"duration": 500,
						"interval": 500,
					},
				},
			},

			"12345",
			500,
			500,
			4500,
		},
		{
			"send 1 digits",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("49a66b38-69c0-11eb-b96c-d799dd21ba8f"),
				},
				ChannelID: "49e625de-69c0-11eb-891d-db5407ae4982",
				Action: fmaction.Action{
					Type: fmaction.TypeDigitsSend,
					ID:   uuid.FromStringOrNil("4a24912a-69c0-11eb-a334-6f8053ede87a"),
					Option: map[string]any{
						"digits":   "1",
						"duration": 500,
						"interval": 500,
					},
				},
			},

			"1",
			500,
			500,
			500,
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

			mockChannel.EXPECT().DTMFSend(ctx, tt.call.ChannelID, tt.expectDigits, tt.expectDuration, 0, tt.expectInterval, 0).Return(nil)
			mockReq.EXPECT().CallV1CallActionTimeout(ctx, tt.call.ID, tt.expectTimeout, &tt.call.Action)

			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteExternalMediaStart(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		responseExternalMedia *externalmedia.ExternalMedia

		expectHost            string
		expectEncapsulation   externalmedia.Encapsulation
		expectTransport       externalmedia.Transport
		expectConnectionType  string
		expectFormat          string
		expectDirectionListen externalmedia.Direction
		expectDirectionSpeak  externalmedia.Direction
	}{
		{
			name: "normal",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3ba00ae0-02f8-11ec-863a-abd78c8246c4"),
				},
				ChannelID: "4455e2f4-02f8-11ec-acf9-43a391fce607",
				Action: fmaction.Action{
					Type: fmaction.TypeExternalMediaStart,
					ID:   uuid.FromStringOrNil("447f0d28-02f8-11ec-bfdb-4bb2407458ce"),
					Option: map[string]any{
						"external_host":    "example.com",
						"encapsulation":    "rtp",
						"transport":        "udp",
						"connection_type":  "client",
						"format":           "ulaw",
						"direction_listen": "in",
						"direction_speak":  "out",
						"data":             "",
					},
				},
			},

			responseExternalMedia: &externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("caff4130-96ed-11ed-a978-ff691cc47d66"),
			},

			expectHost:            "example.com",
			expectEncapsulation:   "rtp",
			expectTransport:       "udp",
			expectConnectionType:  "client",
			expectFormat:          "ulaw",
			expectDirectionListen: externalmedia.DirectionIn,
			expectDirectionSpeak:  externalmedia.DirectionOut,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &callHandler{
				utilHandler:          mockUtil,
				reqHandler:           mockReq,
				db:                   mockDB,
				externalMediaHandler: mockExternal,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockExternal.EXPECT().Start(
				ctx,
				uuid.Nil,
				externalmedia.ReferenceTypeCall,
				tt.call.ID,
				tt.expectHost,
				tt.expectEncapsulation,
				tt.expectTransport,
				tt.expectConnectionType,
				tt.expectFormat,
				tt.expectDirectionListen,
				tt.expectDirectionSpeak,
			).Return(tt.responseExternalMedia, nil)
			mockDB.EXPECT().CallSetExternalMediaID(ctx, tt.call.ID, tt.responseExternalMedia.ID).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)
			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionExecute_actionExecuteExternalMediaStop(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		responseExtMedia *externalmedia.ExternalMedia
	}{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("50b8cb46-1aa5-11ec-9b1e-7b766955c7d1"),
				},
				ChannelID:       "4455e2f4-02f8-11ec-acf9-43a391fce607",
				ExternalMediaID: uuid.FromStringOrNil("c7e222a4-96ef-11ed-a9c7-731c399f5537"),
				Action: fmaction.Action{
					Type: fmaction.TypeExternalMediaStop,
					ID:   uuid.FromStringOrNil("50ff55d4-1aa5-11ec-8d4e-7fc834754547"),
				},
			},

			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("c7e222a4-96ef-11ed-a9c7-731c399f5537"),
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
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &callHandler{
				utilHandler:          mockUtil,
				reqHandler:           mockReq,
				db:                   mockDB,
				externalMediaHandler: mockExternal,
			}

			ctx := context.Background()

			mockExternal.EXPECT().Stop(ctx, tt.call.ExternalMediaID).Return(tt.responseExtMedia, nil)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)

			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionExecuteAMD(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		expectAMD *callapplication.AMD
	}{
		{
			"sync false",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f607e1b2-19b6-11ec-8304-a33ee590d878"),
				},
				ChannelID: "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				Action: fmaction.Action{
					Type: fmaction.TypeAMD,
					ID:   uuid.FromStringOrNil("f681c108-19b6-11ec-bc57-635de4310a4b"),
					Option: map[string]any{
						"machine_handle": "hangup",
					},
				},
			},

			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("f607e1b2-19b6-11ec-8304-a33ee590d878"),
				MachineHandle: "hangup",
			},
		},
		{
			"sync true",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7d89362a-19b9-11ec-a1ea-a74ce01d2c9b"),
				},
				ChannelID: "7da1b4fc-19b9-11ec-948e-7f9ca90957a1",
				Action: fmaction.Action{
					Type: fmaction.TypeAMD,
					ID:   uuid.FromStringOrNil("7dba7df2-19b9-11ec-b426-17e356fbf5e3"),
					Option: map[string]any{
						"machine_handle": "hangup",
						"sync":           true,
					},
				},
			},

			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("7d89362a-19b9-11ec-a1ea-a74ce01d2c9b"),
				MachineHandle: "hangup",
				Async:         false,
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

			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockChannel.EXPECT().StartSnoop(ctx, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirectionBoth, channel.SnoopDirectionBoth).Return(&channel.Channel{}, nil)
			mockDB.EXPECT().CallApplicationAMDSet(ctx, gomock.Any(), tt.expectAMD).Return(nil)

			if tt.expectAMD.Async == true {
				mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)
			}
			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_cleanCurrentAction(t *testing.T) {

	tests := []struct {
		name            string
		call            *call.Call
		responseChannel *channel.Channel
		expectRes       bool
	}{
		{
			"playback has set",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f607e1b2-19b6-11ec-8304-a33ee590d878"),
				},
				ChannelID: "f6593184-19b6-11ec-85ee-8bda2a70f32e",
			},
			&channel.Channel{
				ID:         "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				AsteriskID: "42:01:0a:a4:00:05",
				PlaybackID: "44a07af0-5837-11ec-bdce-6bfc534e86b7",
			},
			false,
		},
		{
			"confbridgeID has set",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f607e1b2-19b6-11ec-8304-a33ee590d878"),
				},
				ChannelID:    "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				ConfbridgeID: uuid.FromStringOrNil("619bba82-5839-11ec-8733-c3a8bf0aee26"),
			},
			&channel.Channel{
				ID:         "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			false,
		},
		{
			"action sleep",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0d074aee-8c1a-11ec-b499-c33db4145901"),
				},
				ChannelID: "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				Action: fmaction.Action{
					Type: fmaction.TypeSleep,
				},
			},
			&channel.Channel{
				ID:         "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockConf := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				confbridgeHandler: mockConf,
				channelHandler:    mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().Get(ctx, tt.call.ChannelID).Return(tt.responseChannel, nil)
			if tt.responseChannel.PlaybackID != "" {
				mockChannel.EXPECT().PlaybackStop(ctx, tt.call.ChannelID).Return(nil)
			}

			if tt.call.ConfbridgeID != uuid.Nil {
				mockConf.EXPECT().Kick(ctx, tt.call.ConfbridgeID, tt.call.ID).Return(nil)
			}

			res, err := h.cleanCurrentAction(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.expectRes != res {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ActionNext(t *testing.T) {

	tests := []struct {
		name           string
		call           *call.Call
		channel        *channel.Channel
		responseAction *fmaction.Action
		responseCall   *call.Call
	}{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f607e1b2-19b6-11ec-8304-a33ee590d878"),
				},
				ChannelID:    "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				Status:       call.StatusProgressing,
				FlowID:       uuid.FromStringOrNil("82beb924-583b-11ec-955a-236e3409cf25"),
				ActiveflowID: uuid.FromStringOrNil("01603928-a7bb-11ec-86d6-57ce9c598437"),
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("c9bc39a0-583b-11ec-b0c4-2373b012eba7"),
					Type: fmaction.TypeTalk,
				},
			},
			&channel.Channel{
				ID:         "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				AsteriskID: "42:01:0a:a4:00:05",
				PlaybackID: "44a07af0-5837-11ec-bdce-6bfc534e86b7",
			},
			&fmaction.Action{
				ID: uuid.FromStringOrNil("fe96418e-583b-11ec-93d8-738261aee2c9"),
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f607e1b2-19b6-11ec-8304-a33ee590d878"),
				},
				ChannelID:    "f6593184-19b6-11ec-85ee-8bda2a70f32e",
				Status:       call.StatusProgressing,
				FlowID:       uuid.FromStringOrNil("82beb924-583b-11ec-955a-236e3409cf25"),
				ActiveflowID: uuid.FromStringOrNil("01603928-a7bb-11ec-86d6-57ce9c598437"),
				Action: fmaction.Action{
					ID: uuid.FromStringOrNil("fe96418e-583b-11ec-93d8-738261aee2c9"),
				},
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallSetActionNextHold(ctx, tt.call.ID, true).Return(nil)
			mockReq.EXPECT().FlowV1ActiveflowGetNextAction(ctx, tt.call.ActiveflowID, tt.call.Action.ID).Return(tt.responseAction, nil)
			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockDB.EXPECT().CallSetActionAndActionNextHold(ctx, tt.call.ID, tt.responseAction, false).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallUpdated, tt.responseCall)

			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)

			if err := h.ActionNext(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionExecuteHangup_reason(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		responseCall    *call.Call
		responseChannel *channel.Channel
		expectCause     ari.ChannelCause
	}{
		{
			name: "normal",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a2da5b15-403f-492d-96e0-53f883028d88"),
				},
				ChannelID: "105567ee-61de-4eed-98d4-d6b0d2667f3a",
				Status:    call.StatusDialing,
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("c2f794a4-0fa5-43b7-a47a-599edcde6b55"),
					Type: fmaction.TypeHangup,
					Option: map[string]any{
						"reason": "busy",
					},
				},
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a2da5b15-403f-492d-96e0-53f883028d88"),
				},
				ChannelID: "105567ee-61de-4eed-98d4-d6b0d2667f3a",
				Status:    call.StatusTerminating,
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("c2f794a4-0fa5-43b7-a47a-599edcde6b55"),
					Type: fmaction.TypeHangup,
					Option: map[string]any{
						"reason": "busy",
					},
				},
			},
			responseChannel: &channel.Channel{
				TMEnd: nil,
			},
			expectCause: ari.ChannelCauseUserBusy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatus(ctx, tt.call.ID, call.StatusTerminating).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallTerminating, tt.responseCall)

			mockChannel.EXPECT().HangingUp(ctx, tt.call.ChannelID, tt.expectCause).Return(tt.responseChannel, nil)

			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionExecuteHangup_reference(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		responseReferenceCall    *call.Call
		responseReferenceChannel *channel.Channel
		responseCall             *call.Call
		responseChannel          *channel.Channel

		expectCause ari.ChannelCause
	}{
		{
			name: "call hungup by user busy",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("12ea8d3e-52b3-4c8e-a46c-a9d66a40c94c"),
				},
				ChannelID: "eeddbb76-4bd8-4aa7-a6fd-c18690474eb6",
				Status:    call.StatusDialing,
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("324ef8af-508d-4622-8f1c-1df75efe70a6"),
					Type: fmaction.TypeHangup,
					Option: map[string]any{
						"reference_id": "94d73f3f-0158-4172-8ffa-5d7a7f2bd8a4",
					},
				},
			},

			responseReferenceCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("94d73f3f-0158-4172-8ffa-5d7a7f2bd8a4"),
				},
				ChannelID:    "f4cf0996-a3d1-11ed-8aca-97c846819d72",
				Status:       call.StatusHangup,
				HangupReason: call.HangupReasonBusy,
			},
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("12ea8d3e-52b3-4c8e-a46c-a9d66a40c94c"),
				},
				ChannelID: "eeddbb76-4bd8-4aa7-a6fd-c18690474eb6",
				Status:    call.StatusTerminating,
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("324ef8af-508d-4622-8f1c-1df75efe70a6"),
					Type: fmaction.TypeHangup,
					Option: map[string]any{
						"reference_id": "94d73f3f-0158-4172-8ffa-5d7a7f2bd8a4",
					},
				},
			},
			responseReferenceChannel: &channel.Channel{
				ID:          "f4cf0996-a3d1-11ed-8aca-97c846819d72",
				HangupCause: ari.ChannelCauseUserBusy,
			},
			responseChannel: &channel.Channel{
				TMEnd: nil,
			},

			expectCause: ari.ChannelCauseUserBusy,
		},
		{
			name: "reason failed with no route destination",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69788128-a3d2-11ed-8721-235dc7d17c81"),
				},
				ChannelID: "6997f17a-a3d2-11ed-a9db-ff2c89c38193",
				Status:    call.StatusDialing,
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("69b5bb6a-a3d2-11ed-b24e-33e293af5d6d"),
					Type: fmaction.TypeHangup,
					Option: map[string]any{
						"reference_id": "69d1ff28-a3d2-11ed-be5d-f3ea54e96121",
					},
				},
			},

			responseReferenceCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69d1ff28-a3d2-11ed-be5d-f3ea54e96121"),
				},
				ChannelID:    "7888691c-a3d2-11ed-9e53-5ba62e8286cb",
				Status:       call.StatusHangup,
				HangupReason: call.HangupReasonFailed,
			},
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69788128-a3d2-11ed-8721-235dc7d17c81"),
				},
				ChannelID: "6997f17a-a3d2-11ed-a9db-ff2c89c38193",
				Status:    call.StatusTerminating,
				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("69b5bb6a-a3d2-11ed-b24e-33e293af5d6d"),
					Type: fmaction.TypeHangup,
					Option: map[string]any{
						"reference_id": "69d1ff28-a3d2-11ed-be5d-f3ea54e96121",
					},
				},
			},
			responseReferenceChannel: &channel.Channel{
				ID:          "7888691c-a3d2-11ed-9e53-5ba62e8286cb",
				HangupCause: ari.ChannelCauseNoRouteDestination,
			},
			responseChannel: &channel.Channel{
				TMEnd: nil,
			},

			expectCause: ari.ChannelCauseNoRouteDestination,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallGet(ctx, tt.responseReferenceCall.ID).Return(tt.responseReferenceCall, nil)
			mockChannel.EXPECT().Get(ctx, tt.responseReferenceCall.ChannelID).Return(tt.responseReferenceChannel, nil)

			mockDB.EXPECT().CallSetStatus(ctx, tt.call.ID, call.StatusTerminating).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallTerminating, tt.responseCall)

			mockChannel.EXPECT().HangingUp(ctx, tt.call.ChannelID, tt.expectCause).Return(tt.responseChannel, nil)

			if err := h.actionExecute(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ActionNextForce(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseChannel *channel.Channel
	}{
		{
			name: "normal",

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d2b36964-264f-11ee-85d8-1fc4aba6d482"),
				},
				ChannelID: "d2eff03c-264f-11ee-be38-dff577e39038",
				Status:    call.StatusProgressing,
			},

			responseChannel: &channel.Channel{
				ID:    "d3234c52-264f-11ee-951a-83d7ae2f2751",
				TMEnd: nil,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallSetActionNextHold(ctx, tt.call.ID, false).Return(nil)
			mockChannel.EXPECT().Get(ctx, tt.call.ChannelID).Return(tt.responseChannel, nil)

			// Hangup
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatus(ctx, tt.call.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.call.CustomerID, gomock.Any(), gomock.Any())
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(tt.responseChannel, nil)

			if err := h.ActionNextForce(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
