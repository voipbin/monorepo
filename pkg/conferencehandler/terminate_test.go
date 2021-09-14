package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestTerminateCallExsist(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := conferenceHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
		cache:         mockCache,
	}

	type test struct {
		name       string
		id         uuid.UUID
		conference *conference.Conference
		bridge     *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
			&conference.Conference{
				ID:   uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
				Type: conference.TypeConference,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("2c4eaf4a-9482-11eb-9c2a-57de7ce9aed1"),
				},
			},
			&bridge.Bridge{
				AsteriskID: "80:fa:5b:5e:da:81",
				ID:         "86918a90-ddc1-11ea-87cb-87d08ecc726f",
				ChannelIDs: []string{
					"640452be-9482-11eb-8fce-cba4c72abe18",
				},
			},
		},
		{
			"2 calls in the conference",
			uuid.FromStringOrNil("fbf41954-0ab4-11eb-a22f-671a43bddb11"),
			&conference.Conference{
				ID:   uuid.FromStringOrNil("fbf41954-0ab4-11eb-a22f-671a43bddb11"),
				Type: conference.TypeConference,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("33a1af9a-9482-11eb-90d1-d7f2cf2288cb"),
					uuid.FromStringOrNil("6dfae364-9482-11eb-b11c-0f47944e2c54"),
				},
			},
			&bridge.Bridge{
				AsteriskID: "80:fa:5b:5e:da:81",
				ID:         "0077b86e-0ab5-11eb-90a5-176365109da1",
				ChannelIDs: []string{
					"03d30630-0ab5-11eb-8313-6b2621c82106",
					"7c769780-9482-11eb-a1db-0b09df4964ef",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.id).Return(tt.conference, nil)
			mockDB.EXPECT().ConferenceSetStatus(gomock.Any(), tt.id, conference.StatusTerminating).Return(nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.conference.BridgeID).Return(tt.bridge, nil)
			for _, id := range tt.bridge.ChannelIDs {
				mockReq.EXPECT().AstChannelHangup(tt.bridge.AsteriskID, id, ari.ChannelCauseNormalClearing).Return(nil)
			}

			if err := h.Terminate(tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})

	}
}

func TestTerminateCallNotExsist(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := conferenceHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
		cache:         mockCache,
	}

	type test struct {
		name       string
		conference *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&conference.Conference{
				ID:   uuid.FromStringOrNil("9f5001a6-9482-11eb-956e-f7ead445bb7a"),
				Type: conference.TypeConference,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			mockDB.EXPECT().ConferenceSetStatus(gomock.Any(), tt.conference.ID, conference.StatusTerminating).Return(nil)

			if err := h.Terminate(tt.conference.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})

	}
}

func TestHangupAllChannelsInBridge(t *testing.T) {
	type test struct {
		name   string
		bridge *bridge.Bridge
	}

	tests := []test{
		{
			"empty channel",
			&bridge.Bridge{
				AsteriskID: "80:fa:5b:5e:da:81",
				ID:         "2593f1bc-9242-11ea-b81e-bf42eb4d93ea",
				ChannelIDs: []string{},
			},
		},
		{
			"1 channel",
			&bridge.Bridge{
				AsteriskID: "80:fa:5b:5e:da:81",
				ID:         "5fed4b3c-9243-11ea-bac0-a3b9c3b6d318",
				ChannelIDs: []string{"633ed328-9243-11ea-8986-8703e25b2ae5"},
			},
		},
		{
			"2 channels",
			&bridge.Bridge{
				AsteriskID: "80:fa:5b:5e:da:81",
				ID:         "7b704f4e-9243-11ea-8051-fb169fb874ad",
				ChannelIDs: []string{"810da186-9243-11ea-980d-a7af8eb709c3", "8587b954-9243-11ea-9b55-a74fe5779f96"},
			},
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, channelID := range tt.bridge.ChannelIDs {
				mockReq.EXPECT().AstChannelHangup(tt.bridge.AsteriskID, channelID, ari.ChannelCauseNormalClearing).Return(nil)
			}
			h.hangupAllChannelsInBridge(tt.bridge)
		})
	}
}
