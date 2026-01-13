package confbridgehandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Leaved(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge

		responseConfbridge *confbridge.Confbridge

		expectConfbridgeID uuid.UUID
		expectCallID       uuid.UUID
	}{
		{
			name: "normal",
			channel: &channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "372b84b4-38e8-11ec-b135-638987bdf59b",
				StasisData: map[channel.StasisDataType]string{
					"confbridge_id": "eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2",
					"call_id":       "eaa09918-38e7-11ec-b386-bb681c4ba744",
				},
			},
			bridge: &bridge.Bridge{
				AsteriskID:    "00:11:22:33:44:55",
				ID:            "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
			},

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
				},
				BridgeID: "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				Type:     confbridge.TypeConference,
				ChannelCallIDs: map[string]uuid.UUID{
					"372b84b4-38e8-11ec-b135-638987bdf59b": uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
					"82d7c562-d6d6-11ec-b40a-8b93a18cec7e": uuid.FromStringOrNil("82ff0e4c-d6d6-11ec-9b01-aba6fd69e457"),
				},
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectConfbridgeID: uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
			expectCallID:       uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
		},
		{
			name: "confbridge connect type has 1 channel",
			channel: &channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "372b84b4-38e8-11ec-b135-638987bdf59b",
				StasisData: map[channel.StasisDataType]string{
					"confbridge_id": "72c6f936-d6d6-11ec-ae21-2f89b16a3e4b",
					"call_id":       "eaa09918-38e7-11ec-b386-bb681c4ba744",
				},
			},
			bridge: &bridge.Bridge{
				AsteriskID:    "00:11:22:33:44:55",
				ID:            "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("72c6f936-d6d6-11ec-ae21-2f89b16a3e4b"),
			},

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72c6f936-d6d6-11ec-ae21-2f89b16a3e4b"),
				},
				BridgeID: "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				Type:     confbridge.TypeConnect,
				ChannelCallIDs: map[string]uuid.UUID{
					"372b84b4-38e8-11ec-b135-638987bdf59b": uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
				},
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectConfbridgeID: uuid.FromStringOrNil("72c6f936-d6d6-11ec-ae21-2f89b16a3e4b"),
			expectCallID:       uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
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

			h := confbridgeHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				cache:          mockCache,
				notifyHandler:  mockNotify,
				bridgeHandler:  mockBridge,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(tt.channel, nil)

			mockDB.EXPECT().ConfbridgeRemoveChannelCallID(ctx, tt.expectConfbridgeID, tt.channel.ID).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeLeaved, gomock.Any())

			mockReq.EXPECT().CallV1CallUpdateConfbridgeID(ctx, tt.expectCallID, uuid.Nil).Return(&call.Call{}, nil)

			if tt.responseConfbridge.Type == confbridge.TypeConnect && len(tt.responseConfbridge.ChannelCallIDs) == 1 && !h.flagExist(ctx, tt.responseConfbridge.Flags, confbridge.FlagNoAutoLeave) {
				// Terminating
				mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridge, nil)
				mockDB.EXPECT().ConfbridgeSetStatus(ctx, tt.responseConfbridge.ID, confbridge.StatusTerminating).Return(nil)
				mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridge, nil)
				mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeTerminating, gomock.Any())

				mockBridge.EXPECT().Destroy(ctx, tt.responseConfbridge.BridgeID).Return(nil)
			}

			if err := h.Leaved(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
