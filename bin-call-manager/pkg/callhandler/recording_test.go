package callhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

func Test_RecordingStart(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		format       recording.Format
		endOfSilence int
		endOfKey     string
		duration     int
		onEndFlowID  uuid.UUID

		responseCall      *call.Call
		responseRecording *recording.Recording

		expectAnswer bool
	}{
		{
			name: "call is already progressing - no auto answer",

			id:           uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
			format:       recording.FormatWAV,
			endOfSilence: 10000,
			endOfKey:     "#",
			duration:     86400,
			onEndFlowID:  uuid.FromStringOrNil("2b26d3b2-0545-11f0-b4f2-9712e07952c5"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
				},
				ActiveflowID: uuid.FromStringOrNil("682ea1e2-0728-11f0-becb-83d82ea88b27"),
				ChannelID:    "channel-progressing",
				Status:       call.StatusProgressing,
			},
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5c2e0aa4-9317-11ed-83be-4bc8dcb3ae1d"),
				},
			},

			expectAnswer: false,
		},
		{
			name: "call is ringing - auto answer then record",

			id:           uuid.FromStringOrNil("b0f8b0d0-91f0-11f0-9d0e-3f9c5b2a7e10"),
			format:       recording.FormatWAV,
			endOfSilence: 10000,
			endOfKey:     "#",
			duration:     86400,
			onEndFlowID:  uuid.FromStringOrNil("b1204d3c-91f0-11f0-8b2a-7f3d6c1a9e22"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b0f8b0d0-91f0-11f0-9d0e-3f9c5b2a7e10"),
				},
				ActiveflowID: uuid.FromStringOrNil("b14a1e88-91f0-11f0-a3b5-6f2c9d4b8e34"),
				ChannelID:    "channel-ringing",
				Status:       call.StatusRinging,
			},
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b1716a2e-91f0-11f0-9c1d-4b8e2f6a3d56"),
				},
			},

			expectAnswer: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				notifyHandler:    mockNotify,
				db:               mockDB,
				recordingHandler: mockRecording,
				channelHandler:   mockChannel,
			}

			ctx := context.Background()
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			if tt.expectAnswer {
				mockChannel.EXPECT().Answer(ctx, tt.responseCall.ChannelID).Return(nil)
			}
			mockRecording.EXPECT().Start(ctx, tt.responseCall.ActiveflowID, recording.ReferenceTypeCall, tt.responseCall.ID, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration, tt.onEndFlowID).Return(tt.responseRecording, nil)
			mockDB.EXPECT().CallSetRecordingID(ctx, tt.responseCall.ID, tt.responseRecording.ID).Return(nil)
			mockDB.EXPECT().CallAddRecordingIDs(ctx, tt.responseCall.ID, tt.responseRecording.ID).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)

			res, err := h.RecordingStart(ctx, tt.id, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration, tt.onEndFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseCall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseCall, res)
			}
		})
	}
}

func Test_RecordingStart_error(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		format       recording.Format
		endOfSilence int
		endOfKey     string
		duration     int
		onEndFlowID  uuid.UUID

		responseCall *call.Call

		expectAnswer      bool
		responseAnswerErr error
	}{
		{
			name: "recording is already progressing - rejected before auto answer even when ringing",

			id:           uuid.FromStringOrNil("c2a1f0aa-91f0-11f0-8f1a-7b3c5d2e9a44"),
			format:       recording.FormatWAV,
			endOfSilence: 10000,
			endOfKey:     "#",
			duration:     86400,
			onEndFlowID:  uuid.FromStringOrNil("c2c93a70-91f0-11f0-9d5e-4f2b8c1a6e55"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c2a1f0aa-91f0-11f0-8f1a-7b3c5d2e9a44"),
				},
				ChannelID:   "channel-already-recording",
				Status:      call.StatusRinging,
				RecordingID: uuid.FromStringOrNil("c2f01b3e-91f0-11f0-8a2c-3d6f9b4e1c66"),
			},

			expectAnswer: false,
		},
		{
			name: "call is ringing but answer fails - no recording started",

			id:           uuid.FromStringOrNil("d3b2e1bb-91f0-11f0-9e2b-8c4d6a3f0b77"),
			format:       recording.FormatWAV,
			endOfSilence: 10000,
			endOfKey:     "#",
			duration:     86400,
			onEndFlowID:  uuid.FromStringOrNil("d3da8c22-91f0-11f0-af3c-5e2b9d1a7c88"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d3b2e1bb-91f0-11f0-9e2b-8c4d6a3f0b77"),
				},
				ChannelID: "channel-hungup",
				Status:    call.StatusRinging,
			},

			expectAnswer:      true,
			responseAnswerErr: errors.New("the channel has hungup already"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				notifyHandler:    mockNotify,
				db:               mockDB,
				recordingHandler: mockRecording,
				channelHandler:   mockChannel,
			}

			ctx := context.Background()
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			if tt.expectAnswer {
				mockChannel.EXPECT().Answer(ctx, tt.responseCall.ChannelID).Return(tt.responseAnswerErr)
			}
			// recordingHandler.Start and the recording-id writes must NOT be reached on
			// the reject/answer-failure paths (strict gomock fails on any unexpected call).

			res, err := h.RecordingStart(ctx, tt.id, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration, tt.onEndFlowID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_RecordingStop(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		format       string
		endOfSilence int
		endOfKey     string
		duration     int

		responseCall      *call.Call
		responseRecording *recording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("9fbc00a0-9317-11ed-b20d-374b334d2c55"),
			"wav",
			10000,
			"#",
			86400,

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9fbc00a0-9317-11ed-b20d-374b334d2c55"),
				},
				Status:      call.StatusProgressing,
				RecordingID: uuid.FromStringOrNil("9fea43b6-9317-11ed-9777-bbde3dec8816"),
			},
			&recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9fea43b6-9317-11ed-9777-bbde3dec8816"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				notifyHandler:    mockNotify,
				db:               mockDB,
				recordingHandler: mockRecording,
			}

			ctx := context.Background()
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockRecording.EXPECT().Stop(ctx, tt.responseCall.RecordingID).Return(tt.responseRecording, nil)
			mockDB.EXPECT().CallSetRecordingID(ctx, tt.responseCall.ID, uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)

			res, err := h.RecordingStop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseCall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseCall, res)
			}
		})
	}
}
