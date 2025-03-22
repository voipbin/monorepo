package confbridgehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Terminating(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConfbridge       *confbridge.Confbridge
		responseConfbridgeUpdate *confbridge.Confbridge
	}{
		{
			name: "have no bridge id",

			id: uuid.FromStringOrNil("947d3525-8bda-4a6c-8167-7579680c334c"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("947d3525-8bda-4a6c-8167-7579680c334c"),
				},
				Type:     confbridge.TypeConnect,
				Status:   confbridge.StatusProgressing,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseConfbridgeUpdate: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("947d3525-8bda-4a6c-8167-7579680c334c"),
				},
				Type:     confbridge.TypeConnect,
				Status:   confbridge.StatusTerminating,
				TMDelete: dbhandler.DefaultTimeStamp},
		},
		{
			name: "have bridge id",

			id: uuid.FromStringOrNil("0e9ad733-f027-4ba3-932f-dede201f3726"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0e9ad733-f027-4ba3-932f-dede201f3726"),
				},
				Type:     confbridge.TypeConnect,
				Status:   confbridge.StatusProgressing,
				BridgeID: "ea17cc48-592d-4054-8424-ead8c3e45a26",
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseConfbridgeUpdate: nil,
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

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)

			mockDB.EXPECT().ConfbridgeSetStatus(ctx, tt.id, confbridge.StatusTerminating).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeTerminating, tt.responseConfbridge)

			if tt.responseConfbridge.BridgeID == "" {
				mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridgeUpdate, nil)
				mockDB.EXPECT().ConfbridgeSetStatus(ctx, tt.responseConfbridge.ID, confbridge.StatusTerminated).Return(nil)
				mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridge, nil)
				mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeTerminated, tt.responseConfbridge)
			} else {
				mockBridge.EXPECT().Destroy(ctx, tt.responseConfbridge.BridgeID).Return(nil)
			}

			res, err := h.Terminating(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConfbridge, res) {
				t.Errorf("Wrong match. expect: %s\ngot:%s\n", tt.responseConfbridge, res)
			}
		})
	}
}

func Test_Terminate(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConfbridge *confbridge.Confbridge
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("31a7c3b7-65ef-4c3c-bd60-3e0c3a27f58b"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31a7c3b7-65ef-4c3c-bd60-3e0c3a27f58b"),
				},
				Type:   confbridge.TypeConnect,
				Status: confbridge.StatusTerminating,
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

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)

			mockDB.EXPECT().ConfbridgeSetStatus(ctx, tt.responseConfbridge.ID, confbridge.StatusTerminated).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeTerminated, tt.responseConfbridge)

			if err := h.Terminate(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
