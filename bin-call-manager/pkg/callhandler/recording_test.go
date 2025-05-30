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
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/recording"
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
	}{
		{
			name: "normal",

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
				Status:       call.StatusProgressing,
			},
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5c2e0aa4-9317-11ed-83be-4bc8dcb3ae1d"),
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
