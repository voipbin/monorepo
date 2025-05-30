package confbridgehandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Kick(t *testing.T) {

	tests := []struct {
		name         string
		confbridgeID uuid.UUID
		callID       uuid.UUID
		confbridge   *confbridge.Confbridge
		channel      *channel.Channel
	}{
		{
			name:         "normal",
			confbridgeID: uuid.FromStringOrNil("ea343f84-38e7-11ec-bcba-df9a707c8d39"),
			callID:       uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),

			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
				},
				BridgeID: "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
				ChannelCallIDs: map[string]uuid.UUID{
					"372b84b4-38e8-11ec-b135-638987bdf59b": uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
				},
			},
			channel: &channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "372b84b4-38e8-11ec-b135-638987bdf59b",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := confbridgeHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				cache:          mockCache,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridgeID).Return(tt.confbridge, nil)
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(tt.channel, nil)

			if err := h.Kick(ctx, tt.confbridgeID, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
