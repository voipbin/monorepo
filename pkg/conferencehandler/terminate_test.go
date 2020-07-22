package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestTerminate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewConferHandler(mockReq, mockDB, mockCache)

	type test struct {
		name       string
		id         uuid.UUID
		conference *conference.Conference
		bridges    []*bridge.Bridge
	}

	tests := []test{
		{
			"empty bridge ids",
			uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
			&conference.Conference{
				ID:        uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
				Type:      conference.TypeEcho,
				BridgeIDs: []string{},
			},
			[]*bridge.Bridge{},
		},
		{
			"1 bridge ids",
			uuid.FromStringOrNil("c33adf04-9240-11ea-a8ed-0fa57555db3b"),
			&conference.Conference{
				ID:        uuid.FromStringOrNil("c33adf04-9240-11ea-a8ed-0fa57555db3b"),
				Type:      conference.TypeEcho,
				BridgeIDs: []string{"c9977272-9240-11ea-a692-f370484c30f1"},
			},
			[]*bridge.Bridge{
				{
					ID:         "c9977272-9240-11ea-a692-f370484c30f1",
					AsteriskID: "00:11:22:33:44:55",
				},
			},
		},
		{
			"2 bridge ids",
			uuid.FromStringOrNil("71a286e6-9241-11ea-8241-874ac5255e40"),
			&conference.Conference{
				ID:        uuid.FromStringOrNil("71a286e6-9241-11ea-8241-874ac5255e40"),
				Type:      conference.TypeEcho,
				BridgeIDs: []string{"76149278-9241-11ea-9ab3-4baddc693eae", "7b67c790-9241-11ea-aebd-332e45a6ebde"},
			},
			[]*bridge.Bridge{
				{
					ID:         "76149278-9241-11ea-9ab3-4baddc693eae",
					AsteriskID: "00:11:22:33:44:55",
				},
				{
					ID:         "7b67c790-9241-11ea-aebd-332e45a6ebde",
					AsteriskID: "00:11:22:33:44:55",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.id).Return(tt.conference, nil)
			mockDB.EXPECT().ConferenceSetStatus(gomock.Any(), tt.id, conference.StatusTerminating).Return(nil)
			for i, bridgeID := range tt.conference.BridgeIDs {
				mockDB.EXPECT().BridgeGetFromCache(gomock.Any(), bridgeID).Return(tt.bridges[i], nil)
				mockReq.EXPECT().AstBridgeDelete(tt.bridges[i].AsteriskID, tt.bridges[i].ID).Return(nil)
			}
			mockDB.EXPECT().ConferenceEnd(gomock.Any(), tt.conference.ID).Return(nil)

			if err := h.Terminate(tt.id, "normal terminating"); err != nil {
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
