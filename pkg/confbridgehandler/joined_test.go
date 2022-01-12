package confbridgehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

func TestJoined(t *testing.T) {
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

	tests := []struct {
		name         string
		channel      *channel.Channel
		bridge       *bridge.Bridge
		callID       uuid.UUID
		confbridgeID uuid.UUID

		confbridge *confbridge.Confbridge
	}{
		{
			"normal",
			&channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "4268f036-38d0-11ec-a912-ebca1cd51965",
				StasisData: map[string]string{
					"confbridge_id": "eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2",
					"call_id":       "ebb3c432-38cf-11ec-ad96-fb9640d4c6ee",
				},
			},
			&bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
				TMDelete:   defaultTimeStamp,
			},
			uuid.FromStringOrNil("ebb3c432-38cf-11ec-ad96-fb9640d4c6ee"),
			uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),

			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
				BridgeID: "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeAddChannelCallID(gomock.Any(), tt.confbridgeID, tt.channel.ID, tt.callID).Return(nil)
			mockDB.EXPECT().CallSetConfbridgeID(gomock.Any(), tt.callID, tt.confbridgeID).Return(nil)

			mockNotify.EXPECT().PublishEvent(gomock.Any(), notifyhandler.EventTypeConfbridgeJoined, gomock.Any())
			mockDB.EXPECT().CallGet(gomock.Any(), tt.callID).Return(&call.Call{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallUpdated, "", gomock.Any())

			if err := h.Joined(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
