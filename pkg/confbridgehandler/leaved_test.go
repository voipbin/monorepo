package confbridgehandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_Leaved(t *testing.T) {

	tests := []struct {
		name       string
		confbridge *confbridge.Confbridge
		callID     uuid.UUID
		channel    *channel.Channel
		bridge     *bridge.Bridge
	}{
		{
			"normal",
			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
				BridgeID: "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				Type:     confbridge.TypeConference,
				ChannelCallIDs: map[string]uuid.UUID{
					"372b84b4-38e8-11ec-b135-638987bdf59b": uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
					"82d7c562-d6d6-11ec-b40a-8b93a18cec7e": uuid.FromStringOrNil("82ff0e4c-d6d6-11ec-9b01-aba6fd69e457"),
				},
			},
			uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
			&channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "372b84b4-38e8-11ec-b135-638987bdf59b",
				StasisData: map[string]string{
					"confbridge_id": "eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2",
					"call_id":       "eaa09918-38e7-11ec-b386-bb681c4ba744",
				},
			},
			&bridge.Bridge{
				AsteriskID:    "00:11:22:33:44:55",
				ID:            "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
			},
		},
		{
			"confbridge has 1 channel",
			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("72c6f936-d6d6-11ec-ae21-2f89b16a3e4b"),
				BridgeID: "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				Type:     confbridge.TypeConnect,
				ChannelCallIDs: map[string]uuid.UUID{
					"372b84b4-38e8-11ec-b135-638987bdf59b": uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
				},
			},
			uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
			&channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "372b84b4-38e8-11ec-b135-638987bdf59b",
				StasisData: map[string]string{
					"confbridge_id": "72c6f936-d6d6-11ec-ae21-2f89b16a3e4b",
					"call_id":       "eaa09918-38e7-11ec-b386-bb681c4ba744",
				},
			},
			&bridge.Bridge{
				AsteriskID:    "00:11:22:33:44:55",
				ID:            "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("72c6f936-d6d6-11ec-ae21-2f89b16a3e4b"),
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

			h := confbridgeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeRemoveChannelCallID(ctx, tt.confbridge.ID, tt.channel.ID).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID).Return(tt.confbridge, nil)

			mockDB.EXPECT().CallSetConfbridgeID(ctx, tt.callID, uuid.Nil).Return(nil)

			if tt.confbridge.Type == confbridge.TypeConnect && len(tt.confbridge.ChannelCallIDs) == 1 {
				for _, joinedCallID := range tt.confbridge.ChannelCallIDs {
					mockReq.EXPECT().CallV1ConfbridgeCallKick(ctx, tt.confbridge.ID, joinedCallID).Return(nil)
				}
			}

			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeLeaved, gomock.Any())
			mockDB.EXPECT().CallGet(ctx, tt.callID).Return(&call.Call{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

			if err := h.Leaved(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
