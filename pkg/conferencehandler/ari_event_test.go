package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
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

	h := &conferenceHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		cache:         mockCache,
	}

	tests := []struct {
		name    string
		channel *channel.Channel
		data    map[string]interface{}
	}{
		{
			"conference incoming",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
			},
			map[string]interface{}{
				"context": contextConferenceIncoming,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeConf)).Return(nil)
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

	h := &conferenceHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		cache:         mockCache,
	}

	tests := []struct {
		name    string
		channel *channel.Channel
		data    map[string]interface{}
	}{
		{
			"conference outgoing",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
			},
			map[string]interface{}{
				"context": contextConferenceOutgoing,
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
			map[string]interface{}{},
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

func TestARIChannelLeftBridgeConference(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &conferenceHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		cache:         mockCache,
	}

	type test struct {
		name       string
		channel    *channel.Channel
		bridge     *bridge.Bridge
		conference *conference.Conference
		call       *call.Call
	}

	tests := []test{
		{
			"conference left",
			&channel.Channel{
				ID:         "e03dc034-9566-11ea-ad83-1f7a1993587b",
				AsteriskID: "80:fa:5b:5e:da:81",
				Data:       map[string]interface{}{},
				Type:       channel.TypeConf,
			},
			&bridge.Bridge{
				ID:            "e41948fe-9566-11ea-a4fe-db788b6b6d7b",
				ReferenceID:   uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				ReferenceType: bridge.ReferenceTypeConference,
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				Type:     conference.TypeConference,
				Status:   conference.StatusTerminating,
				BridgeID: "e41948fe-9566-11ea-a4fe-db788b6b6d7b",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("ec4371b2-9566-11ea-bfd3-13a7a033d235"),
				ConfID:     uuid.FromStringOrNil("454cb52a-9567-11ea-91be-3b3c3d7249b6"),
				ChannelID:  "e03dc034-9566-11ea-ad83-1f7a1993587b",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ReferenceID).Return(tt.conference, nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ReferenceID).Return(tt.conference, nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(tt.bridge, nil)
			mockReq.EXPECT().AstBridgeDelete(tt.bridge.AsteriskID, tt.bridge.ID).Return(nil)
			mockDB.EXPECT().ConferenceEnd(gomock.Any(), tt.conference.ID).Return(nil)

			err := h.ARIChannelLeftBridge(tt.channel, tt.bridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelLeftBridgeConnect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &conferenceHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		cache:         mockCache,
	}

	type test struct {
		name       string
		channel    *channel.Channel
		bridge     *bridge.Bridge
		conference *conference.Conference
		call       *call.Call
	}

	tests := []test{
		{
			"1 channel still remains in the bridge",
			&channel.Channel{
				ID:         "e03dc034-9566-11ea-ad83-1f7a1993587b",
				AsteriskID: "80:fa:5b:5e:da:81",
				Data:       map[string]interface{}{},
				Type:       channel.TypeConf,
			},
			&bridge.Bridge{
				ID:            "e41948fe-9566-11ea-a4fe-db788b6b6d7b",
				ReferenceID:   uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				ReferenceType: bridge.ReferenceTypeConference,
				ChannelIDs: []string{
					"423d2ffa-16a7-11ec-9214-e39d509d8fa3",
				},
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				Type:     conference.TypeConnect,
				Status:   conference.StatusProgressing,
				BridgeID: "e41948fe-9566-11ea-a4fe-db788b6b6d7b",
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("ea6e8010-16a8-11ec-83eb-c32797acd5dc"),
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("ec4371b2-9566-11ea-bfd3-13a7a033d235"),
				ConfID:     uuid.FromStringOrNil("454cb52a-9567-11ea-91be-3b3c3d7249b6"),
				ChannelID:  "e03dc034-9566-11ea-ad83-1f7a1993587b",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
		},
		{
			"no channel left in the bridge",
			&channel.Channel{
				ID:         "cd898f0c-16a9-11ec-8a3c-07a02c763fc3",
				AsteriskID: "80:fa:5b:5e:da:81",
				Data:       map[string]interface{}{},
				Type:       channel.TypeConf,
			},
			&bridge.Bridge{
				ID:            "cdcba946-16a9-11ec-9db6-fb23e2577d80",
				ReferenceID:   uuid.FromStringOrNil("cdaba236-16a9-11ec-87ef-87cdd6c6b868"),
				ReferenceType: bridge.ReferenceTypeConference,
				ChannelIDs:    []string{},
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("cdaba236-16a9-11ec-87ef-87cdd6c6b868"),
				Type:     conference.TypeConnect,
				Status:   conference.StatusProgressing,
				BridgeID: "cdcba946-16a9-11ec-9db6-fb23e2577d80",
				CallIDs:  []uuid.UUID{},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("ec4371b2-9566-11ea-bfd3-13a7a033d235"),
				ConfID:     uuid.FromStringOrNil("cdaba236-16a9-11ec-87ef-87cdd6c6b868"),
				ChannelID:  "cd898f0c-16a9-11ec-8a3c-07a02c763fc3",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ReferenceID).Return(tt.conference, nil)

			if len(tt.bridge.ChannelIDs) > 0 {
				mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
				mockDB.EXPECT().ConferenceSetStatus(gomock.Any(), tt.conference.ID, conference.StatusTerminating).Return(nil)
				mockDB.EXPECT().BridgeGet(gomock.Any(), tt.conference.BridgeID).Return(tt.bridge, nil)
				for _, channelID := range tt.bridge.ChannelIDs {
					mockReq.EXPECT().AstChannelHangup(tt.bridge.AsteriskID, channelID, ari.ChannelCauseNormalClearing).Return(nil)
				}
			} else {
				mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
				mockDB.EXPECT().BridgeGet(gomock.Any(), tt.conference.BridgeID).Return(tt.bridge, nil)
				mockReq.EXPECT().AstBridgeDelete(tt.bridge.AsteriskID, tt.bridge.ID).Return(nil)
				mockDB.EXPECT().ConferenceEnd(gomock.Any(), tt.conference.ID).Return(nil)
			}

			err := h.ARIChannelLeftBridge(tt.channel, tt.bridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
