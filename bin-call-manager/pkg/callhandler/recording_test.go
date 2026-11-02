package callhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

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

				// the status flips to progressing asynchronously: the first poll still sees
				// ringing, the second sees progressing. The recording starts only after that.
				stillRinging := *tt.responseCall
				progressing := *tt.responseCall
				progressing.Status = call.StatusProgressing
				gomock.InOrder(
					mockDB.EXPECT().CallGet(ctx, tt.id).Return(&stillRinging, nil),
					mockDB.EXPECT().CallGet(ctx, tt.id).Return(&progressing, nil),
				)
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
		staysRinging      bool
		becomesStatus     call.Status
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
		{
			name: "call is answered but never becomes progressing - no recording started",

			id:           uuid.FromStringOrNil("e4c3f2cc-91f0-11f0-8f3c-9d5e7b4a1c99"),
			format:       recording.FormatWAV,
			endOfSilence: 10000,
			endOfKey:     "#",
			duration:     86400,
			onEndFlowID:  uuid.FromStringOrNil("e4eb9d33-91f0-11f0-b04d-6f3c8e2b5daa"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e4c3f2cc-91f0-11f0-8f3c-9d5e7b4a1c99"),
				},
				ChannelID: "channel-never-progressing",
				Status:    call.StatusRinging,
			},

			expectAnswer: true,
			staysRinging: true,
		},
		{
			name: "call is answered but is terminating - no recording started",

			id:           uuid.FromStringOrNil("f5d4a3dd-91f0-11f0-8f3c-9d5e7b4a1caa"),
			format:       recording.FormatWAV,
			endOfSilence: 10000,
			endOfKey:     "#",
			duration:     86400,
			onEndFlowID:  uuid.FromStringOrNil("f5fcae44-91f0-11f0-b04d-6f3c8e2b5dbb"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f5d4a3dd-91f0-11f0-8f3c-9d5e7b4a1caa"),
				},
				ChannelID: "channel-terminating",
				Status:    call.StatusRinging,
			},

			expectAnswer:  true,
			staysRinging:  true,
			becomesStatus: call.StatusTerminating,
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
			if tt.staysRinging {
				// shorten the wait so the test does not sleep for the production timeout.
				waitTimeout, waitInterval := recordingStartAnswerWaitTimeout, recordingStartAnswerWaitInterval
				recordingStartAnswerWaitTimeout, recordingStartAnswerWaitInterval = 50*time.Millisecond, 10*time.Millisecond
				defer func() {
					recordingStartAnswerWaitTimeout, recordingStartAnswerWaitInterval = waitTimeout, waitInterval
				}()

				polled := *tt.responseCall
				if tt.becomesStatus != "" {
					// the call is ending: the wait fails at once instead of timing out.
					polled.Status = tt.becomesStatus
					mockDB.EXPECT().CallGet(ctx, tt.id).Return(&polled, nil).Times(1)
				} else {
					mockDB.EXPECT().CallGet(ctx, tt.id).Return(&polled, nil).MinTimes(1)
				}
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

func Test_RecordingStop_error(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		responseCall *call.Call
	}{
		{
			name: "no recording is progressing - an error is returned, not (nil, nil)",
			id:   uuid.FromStringOrNil("a7e0d0a0-9317-11ed-b20d-374b334d2c01"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a7e0d0a0-9317-11ed-b20d-374b334d2c01"),
				},
				Status: call.StatusProgressing,
			},
		},
		{
			name: "call is not progressing",
			id:   uuid.FromStringOrNil("a7e0d0a0-9317-11ed-b20d-374b334d2c02"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a7e0d0a0-9317-11ed-b20d-374b334d2c02"),
				},
				Status:      call.StatusRinging,
				RecordingID: uuid.FromStringOrNil("a7e0d0a1-9317-11ed-9777-bbde3dec8802"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &callHandler{
				db:               mockDB,
				recordingHandler: mockRecording,
			}

			ctx := context.Background()
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)

			res, err := h.RecordingStop(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_UnsetRecordingID(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		recordingID uuid.UUID

		responseUnset bool
		responseErr   error

		expectErr bool
	}{
		{
			name:          "cleared",
			id:            uuid.FromStringOrNil("b8f1e1b0-9317-11ed-b20d-374b334d2c01"),
			recordingID:   uuid.FromStringOrNil("b8f1e1b1-9317-11ed-9777-bbde3dec8801"),
			responseUnset: true,
		},
		{
			name:          "recording id differs - nothing cleared, not an error",
			id:            uuid.FromStringOrNil("b8f1e1b0-9317-11ed-b20d-374b334d2c02"),
			recordingID:   uuid.FromStringOrNil("b8f1e1b1-9317-11ed-9777-bbde3dec8802"),
			responseUnset: false,
		},
		{
			name:        "db error",
			id:          uuid.FromStringOrNil("b8f1e1b0-9317-11ed-b20d-374b334d2c03"),
			recordingID: uuid.FromStringOrNil("b8f1e1b1-9317-11ed-9777-bbde3dec8803"),
			responseErr: errors.New("db failure"),
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &callHandler{db: mockDB}

			ctx := context.Background()
			mockDB.EXPECT().CallUnsetRecordingIDIfMatch(ctx, tt.id, tt.recordingID).Return(tt.responseUnset, tt.responseErr)

			err := h.UnsetRecordingID(ctx, tt.id, tt.recordingID)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_waitCallProgressing_contextCanceled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &callHandler{db: mockDB}

	id := uuid.FromStringOrNil("a9b8c7d6-91f0-11f0-8f3c-9d5e7b4a1cbb")
	ringing := &call.Call{
		Identity: commonidentity.Identity{ID: id},
		Status:   call.StatusRinging,
	}

	ctx, cancel := context.WithCancel(context.Background())
	mockDB.EXPECT().CallGet(gomock.Any(), id).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*call.Call, error) {
		// the caller gives up while the call is still ringing.
		cancel()
		return ringing, nil
	}).Times(1)

	err := h.waitCallProgressing(ctx, id)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Wrong match. expect: context.Canceled, got: %v", err)
	}
}
