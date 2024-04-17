package confbridgehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
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
			"normal",
			uuid.FromStringOrNil("ea343f84-38e7-11ec-bcba-df9a707c8d39"),
			uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),

			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
				BridgeID: "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
				ChannelCallIDs: map[string]uuid.UUID{
					"372b84b4-38e8-11ec-b135-638987bdf59b": uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
				},
			},
			&channel.Channel{
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
