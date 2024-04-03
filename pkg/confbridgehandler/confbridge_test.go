package confbridgehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
)

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConfbridge *confbridge.Confbridge
		// responseConfbridgeUpdate *confbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("947d3525-8bda-4a6c-8167-7579680c334c"),

			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("947d3525-8bda-4a6c-8167-7579680c334c"),
				Type:     confbridge.TypeConnect,
				Status:   confbridge.StatusTerminating,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			// Terminating
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)

			// dbDelete
			mockDB.EXPECT().ConfbridgeDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeDeleted, tt.responseConfbridge)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConfbridge, res) {
				t.Errorf("Wrong match. expect: %s\ngot:%s\n", tt.responseConfbridge, res)
			}
		})
	}
}
