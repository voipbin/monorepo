package confbridgehandler

import (
	"context"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
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

		responseConfbridge *confbridge.Confbridge
		responseRecording  *recording.Recording
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("2c14e1a2-0544-11f0-9e4a-130f7b7aedd4"),
			format:       recording.FormatWAV,
			endOfSilence: 5,
			endOfKey:     "1",
			duration:     60,
			onEndFlowID:  uuid.FromStringOrNil("2c5dcd04-0544-11f0-9d09-e3e0a5a79726"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2c14e1a2-0544-11f0-9e4a-130f7b7aedd4"),
				},
				ActiveflowID: uuid.FromStringOrNil("67e5a500-0728-11f0-86ab-cb19621e1dd9"),
				Status:       confbridge.StatusProgressing,
				TMDelete:     dbhandler.DefaultTimeStamp,
			},
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2c8c2924-0544-11f0-9c71-ff0bd49ff7fd"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := confbridgeHandler{
				reqHandler:       mockReq,
				db:               mockDB,
				cache:            mockCache,
				notifyHandler:    mockNotify,
				bridgeHandler:    mockBridge,
				channelHandler:   mockChannel,
				recordingHandler: mockRecording,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id.Return(tt.responseConfbridge, nil)
			mockRecording.EXPECT().Start(ctx, tt.responseConfbridge.ActiveflowID, recording.ReferenceTypeConfbridge, tt.id, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration, tt.onEndFlowID.Return(tt.responseRecording, nil)
			mockDB.EXPECT().ConfbridgeSetRecordingID(ctx, tt.id, tt.responseRecording.ID.Return(nil)
			mockDB.EXPECT().ConfbridgeAddRecordingIDs(ctx, tt.id, tt.responseRecording.ID.Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id.Return(tt.responseConfbridge, nil)

			res, err := h.RecordingStart(ctx, tt.id, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration, tt.onEndFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConfbridge, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseConfbridge, res)
			}
		})
	}
}
