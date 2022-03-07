package confbridgehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_ARIStasisStartTypeConference(t *testing.T) {
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
			"normal",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
			},
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("69e97312-3748-11ec-a94b-2357c957d67e"),
				Type:           confbridge.TypeConference,
				BridgeID:       "80c2e1ae-3748-11ec-b52c-c7e704ec1140",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
			map[string]string{
				"context":       contextConfbridgeIncoming,
				"confbridge_id": "69e97312-3748-11ec-a94b-2357c957d67e",
				"call_id":       "22df7716-34f3-11ec-a0d1-1faed65f6fd4",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeConfbridge)).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID).Return(tt.confbridge, nil)
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, tt.channel.DestinationNumber, tt.channel.ID, "", false, false).Return(nil)

			mockReq.EXPECT().AstChannelAnswer(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID).Return(nil)

			err := h.ARIStasisStart(ctx, tt.channel, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ARIStasisStartTypeConferenceError(t *testing.T) {
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
			mockReq.EXPECT().AstChannelHangup(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, ari.ChannelCauseNoRouteDestination, 0).Return(nil)

			if err := h.ARIStasisStart(context.Background(), tt.channel, tt.data); err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_ARIStasisStartTypeConnect(t *testing.T) {
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
			"call already exist",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "d93fdf46-977c-11ec-8403-5b0f71484cde",
			},
			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("d9d609ee-977c-11ec-9a6c-af64f3e8859b"),
				Type:     confbridge.TypeConnect,
				BridgeID: "d93fdf46-977c-11ec-8403-5b0f71484cde",
				ChannelCallIDs: map[string]uuid.UUID{
					"a6b4bf78-9741-11ec-a688-73fbff5fe5e8": uuid.FromStringOrNil("a70ee066-9741-11ec-baba-ebfa678d2fce"),
				},
				RecordingIDs: []uuid.UUID{},
			},
			map[string]string{
				"context":       contextConfbridgeIncoming,
				"confbridge_id": "d9d609ee-977c-11ec-9a6c-af64f3e8859b",
				"call_id":       "d9ff2afe-977c-11ec-8b60-ffaa217bff41",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeConfbridge)).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID).Return(tt.confbridge, nil)
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, tt.channel.DestinationNumber, tt.channel.ID, "", false, false).Return(nil)

			if len(tt.confbridge.ChannelCallIDs) > 0 {
				mockReq.EXPECT().AstChannelAnswer(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID).Return(nil)

				for channelID := range tt.confbridge.ChannelCallIDs {
					mockReq.EXPECT().AstChannelAnswer(ctx, tt.channel.AsteriskID, channelID).Return(nil)
				}
			}

			err := h.ARIStasisStart(ctx, tt.channel, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
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
		name         string
		confbridgeID uuid.UUID
		callID       uuid.UUID
		channel      *channel.Channel
		bridge       *bridge.Bridge
		event        *confbridge.EventConfbridgeJoinedLeaved
	}

	tests := []test{
		{
			"confbridge left",
			uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
			uuid.FromStringOrNil("ef83edb2-3bf9-11ec-bc7d-1f524326656b"),
			&channel.Channel{
				ID:         "e03dc034-9566-11ea-ad83-1f7a1993587b",
				AsteriskID: "80:fa:5b:5e:da:81",
				Data:       map[string]interface{}{},
				StasisData: map[string]string{
					"confbridge_id": "e9051ac8-9566-11ea-bde6-331b8236a4c2",
					"call_id":       "ef83edb2-3bf9-11ec-bc7d-1f524326656b",
				},
				Type: channel.TypeConfbridge,
			},
			&bridge.Bridge{
				ID:            "e41948fe-9566-11ea-a4fe-db788b6b6d7b",
				ReferenceID:   uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				ReferenceType: bridge.ReferenceTypeConfbridge,
			},
			&confbridge.EventConfbridgeJoinedLeaved{
				ID:     uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				CallID: uuid.FromStringOrNil("ef83edb2-3bf9-11ec-bc7d-1f524326656b"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConfbridgeRemoveChannelCallID(gomock.Any(), tt.confbridgeID, tt.channel.ID)
			mockDB.EXPECT().CallSetConfbridgeID(gomock.Any(), tt.callID, uuid.Nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), confbridge.EventTypeConfbridgeLeaved, tt.event)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.callID).Return(&call.Call{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

			err := h.ARIChannelLeftBridge(context.Background(), tt.channel, tt.bridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
