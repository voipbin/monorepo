package confbridgehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestARIStasisStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &confbridgeHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		cache:         mockCache,
	}

	tests := []struct {
		name       string
		channel    *channel.Channel
		confbridge *confbridge.Confbridge
		data       map[string]string
	}{
		{
			"confbridge incoming",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
			},
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("69e97312-3748-11ec-a94b-2357c957d67e"),
				ConferenceID:   uuid.FromStringOrNil("76981f96-3748-11ec-b34f-83dea76e6f0d"),
				BridgeID:       "80c2e1ae-3748-11ec-b52c-c7e704ec1140",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
			map[string]string{
				"context":       contextConfbridgeIncoming,
				"confbridge_id": "15f5582c-34f3-11ec-a88e-07fb1716f396",
				"call_id":       "22df7716-34f3-11ec-a0d1-1faed65f6fd4",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeConfbridge)).Return(nil)
			mockReq.EXPECT().AstChannelAnswer(tt.channel.AsteriskID, tt.channel.ID).Return(nil)
			mockReq.EXPECT().AstBridgeAddChannel(tt.channel.AsteriskID, tt.channel.DestinationNumber, tt.channel.ID, "", false, false).Return(nil)

			err := h.ARIStasisStart(tt.channel, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
func TestARIStasisStartError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &confbridgeHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		cache:         mockCache,
	}

	tests := []struct {
		name    string
		channel *channel.Channel
		data    map[string]string
	}{
		{
			"conference outgoing",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
			},
			map[string]string{
				"context": contextConfbridgeOutgoing,
			},
		},
		{
			"no context",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
			},
			map[string]string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelHangup(tt.channel.AsteriskID, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(nil)

			if err := h.ARIStasisStart(tt.channel, tt.data); err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func TestARIChannelLeftBridgeConfbridge(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &confbridgeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		cache:         mockCache,
	}

	type test struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge
	}

	tests := []test{
		{
			"confbridge left",
			&channel.Channel{
				ID:         "e03dc034-9566-11ea-ad83-1f7a1993587b",
				AsteriskID: "80:fa:5b:5e:da:81",
				Data:       map[string]interface{}{},
				Type:       channel.TypeConfbridge,
			},
			&bridge.Bridge{
				ID:            "e41948fe-9566-11ea-a4fe-db788b6b6d7b",
				ReferenceID:   uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				ReferenceType: bridge.ReferenceTypeConfbridge,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConfbridgeRemoveChannelCallID(gomock.Any(), tt.bridge.ReferenceID, tt.channel.ID)
			mockDB.EXPECT().ConfbridgeGet(gomock.Any(), tt.bridge.ReferenceID).Return(&confbridge.Confbridge{}, nil)
			mockNotify.EXPECT().NotifyEvent(notifyhandler.EventTypeConfbridgeLeaved, "", gomock.Any())

			err := h.ARIChannelLeftBridge(tt.channel, tt.bridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
