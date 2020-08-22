package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/bridge"
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
		bridge     *bridge.Bridge
	}

	tests := []test{
		{
			"empty channels in the bridge",
			uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
			&conference.Conference{
				ID:   uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
				Type: conference.TypeEcho,
			},
			&bridge.Bridge{
				AsteriskID: "80:fa:5b:5e:da:81",
				ID:         "86918a90-ddc1-11ea-87cb-87d08ecc726f",
			},
		},
		{
			"1 channel in the bridge",
			uuid.FromStringOrNil("c33adf04-9240-11ea-a8ed-0fa57555db3b"),
			&conference.Conference{
				ID:   uuid.FromStringOrNil("c33adf04-9240-11ea-a8ed-0fa57555db3b"),
				Type: conference.TypeEcho,
			},
			&bridge.Bridge{
				AsteriskID: "80:fa:5b:5e:da:81",
				ID:         "96b34a94-ddc1-11ea-9cec-0fed666dd9a0",
				ChannelIDs: []string{"ed33a850-ddc1-11ea-9fec-77bf9bede781"},
			},
		},
		{
			"2 bridge ids",
			uuid.FromStringOrNil("71a286e6-9241-11ea-8241-874ac5255e40"),
			&conference.Conference{
				ID:   uuid.FromStringOrNil("71a286e6-9241-11ea-8241-874ac5255e40"),
				Type: conference.TypeEcho,
			},
			&bridge.Bridge{
				AsteriskID: "80:fa:5b:5e:da:81",
				ID:         "93927722-ddc1-11ea-9d16-9be45d7d613d",
				ChannelIDs: []string{"f5ddfac8-ddc1-11ea-8ec9-97793f123bb6", "f26014c6-ddc1-11ea-8b4f-939795e34a19"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.id).Return(tt.conference, nil)
			mockDB.EXPECT().ConferenceSetStatus(gomock.Any(), tt.id, conference.StatusTerminating).Return(nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.conference.BridgeID).Return(tt.bridge, nil)
			for _, id := range tt.bridge.ChannelIDs {
				mockReq.EXPECT().AstBridgeRemoveChannel(tt.bridge.AsteriskID, tt.bridge.ID, id)
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
