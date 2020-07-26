package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestCreateEndpointTarget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
		cache:      mockCache,
	}

	type test struct {
		name            string
		asteriskAddress string
		conference      *conference.Conference
		bridge          *bridge.Bridge
		expectEndpoint  string
	}

	tests := []test{
		{
			"normal",
			"10.10.10.10",
			&conference.Conference{
				ID:       uuid.FromStringOrNil("b8e31aaa-9f18-11ea-b686-8f8d34dbf7ba"),
				BridgeID: "cd62c3f4-9f18-11ea-bf06-036362c26ce3",
			},
			&bridge.Bridge{
				ID:         "cd62c3f4-9f18-11ea-bf06-036362c26ce3",
				AsteriskID: "00:11:22:33:44:55",
			},
			"PJSIP/conf-join/sip:cd62c3f4-9f18-11ea-bf06-036362c26ce3@10.10.10.10:5060",
		},
	}

	for _, tt := range tests {

		mockDB.EXPECT().BridgeGet(gomock.Any(), tt.conference.BridgeID).Return(tt.bridge, nil)
		mockCache.EXPECT().AsteriskAddressInternerGet(gomock.Any(), tt.bridge.AsteriskID).Return(tt.asteriskAddress, nil)

		res, err := h.createEndpointTarget(context.Background(), tt.conference)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if res != tt.expectEndpoint {
			t.Errorf("Wrong match. expect: %s\ngot:%s\n", tt.expectEndpoint, res)
		}
	}
}

func TestJoin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
		cache:      mockCache,
	}

	type test struct {
		name             string
		conference       *conference.Conference
		call             *call.Call
		bridgeJoining    *bridge.Bridge
		birdgeConference *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			&conference.Conference{
				ID:       uuid.FromStringOrNil("89856980-9f1c-11ea-a2e8-272863862e18"),
				Type:     conference.TypeConference,
				BridgeID: "f8057224-9f1c-11ea-93f2-673d5005610b",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("00b8301e-9f1d-11ea-a08e-038ccf4318cd"),
				AsteriskID: "00:11:22:33:44:55",
				ChannelID:  "15feb3f8-9f1d-11ea-a707-7f644f1ae186",
				Type:       call.TypeConference,
			},
			&bridge.Bridge{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "838fddba-9f1e-11ea-9bbd-13f88f4d5d25",
			},
			&bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "48f9992e-9f1f-11ea-84f7-f39855ff99ee",
			},
		},
	}

	for _, tt := range tests {
		mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
		mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
		mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
		mockReq.EXPECT().AstBridgeCreate(tt.call.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(nil)
		mockReq.EXPECT().AstBridgeAddChannel(tt.call.AsteriskID, gomock.Any(), tt.call.ChannelID, "", false, false).Return(nil)
		mockDB.EXPECT().BridgeGet(gomock.Any(), gomock.Any()).Return(tt.bridgeJoining, nil)
		mockCache.EXPECT().AsteriskAddressInternerGet(gomock.Any(), gomock.Any()).Return("", nil)
		mockReq.EXPECT().AstChannelCreate(tt.bridgeJoining.AsteriskID, gomock.Any(), gomock.Any(), gomock.Any(), "", "", gomock.Any()).Return(nil)

		if err := h.Join(tt.conference.ID, tt.call.ID); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}
