package confbridgehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestLeaved(t *testing.T) {
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
		name       string
		confbridge *confbridge.Confbridge
		callID     uuid.UUID
		channel    *channel.Channel
		bridge     *bridge.Bridge
	}{
		{
			"normal",
			&confbridge.Confbridge{
				ID:           uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
				BridgeID:     "1f940122-38e9-11ec-a25c-cb08db10a7c1",
				ConferenceID: uuid.FromStringOrNil("eb93c4ac-38cf-11ec-bcc5-031b06ff96b3"),
				ChannelCallIDs: map[string]uuid.UUID{
					"372b84b4-38e8-11ec-b135-638987bdf59b": uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
				},
			},
			uuid.FromStringOrNil("eaa09918-38e7-11ec-b386-bb681c4ba744"),
			&channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "372b84b4-38e8-11ec-b135-638987bdf59b",
				StasisData: map[string]string{
					"confbridge_id": "eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2",
					"call_id":       "eaa09918-38e7-11ec-b386-bb681c4ba744",
					"conference_id": "eb93c4ac-38cf-11ec-bcc5-031b06ff96b3",
				},
			},
			&bridge.Bridge{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "1f940122-38e9-11ec-a25c-cb08db10a7c1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeRemoveChannelCallID(gomock.Any(), tt.confbridge.ID, tt.channel.ID).Return(nil)
			mockDB.EXPECT().CallSetConferenceID(gomock.Any(), tt.callID, uuid.Nil).Return(nil)
			mockNotify.EXPECT().PublishEvent(notifyhandler.EventTypeConfbridgeLeaved, gomock.Any())
			mockDB.EXPECT().CallGet(gomock.Any(), tt.callID).Return(&call.Call{}, nil)
			mockNotify.EXPECT().NotifyEvent(notifyhandler.EventTypeCallUpdated, "", gomock.Any())

			if err := h.Leaved(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
