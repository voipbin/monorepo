package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
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
		mockCache.EXPECT().AsteriskAddressInternalGet(gomock.Any(), tt.bridge.AsteriskID).Return(tt.asteriskAddress, nil)

		res, err := h.createEndpointTarget(context.Background(), tt.conference)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if res != tt.expectEndpoint {
			t.Errorf("Wrong match. expect: %s\ngot:%s\n", tt.expectEndpoint, res)
		}
	}
}

// func TestJoin(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockReq := requesthandler.NewMockRequestHandler(mc)
// 	mockDB := dbhandler.NewMockDBHandler(mc)
// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	h := conferenceHandler{
// 		reqHandler: mockReq,
// 		db:         mockDB,
// 		cache:      mockCache,
// 	}

// 	type test struct {
// 		name             string
// 		conference       *conference.Conference
// 		call             *call.Call
// 		bridgeJoining    *bridge.Bridge
// 		bridgeConference *bridge.Bridge
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&conference.Conference{
// 				ID:       uuid.FromStringOrNil("89856980-9f1c-11ea-a2e8-272863862e18"),
// 				Type:     conference.TypeConference,
// 				BridgeID: "a5c525ec-dca0-11ea-b139-17780451d9da",
// 			},
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("00b8301e-9f1d-11ea-a08e-038ccf4318cd"),
// 				AsteriskID: "00:11:22:33:44:55",
// 				ChannelID:  "15feb3f8-9f1d-11ea-a707-7f644f1ae186",
// 				Type:       call.TypeConference,
// 			},
// 			&bridge.Bridge{
// 				AsteriskID: "00:11:22:33:44:55",
// 				ID:         "838fddba-9f1e-11ea-9bbd-13f88f4d5d25",
// 			},
// 			&bridge.Bridge{
// 				AsteriskID: "00:11:22:33:44:66",
// 				ID:         "a5c525ec-dca0-11ea-b139-17780451d9da",
// 				TMDelete:   defaultTimeStamp,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
// 		mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
// 		mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
// 		mockDB.EXPECT().BridgeGet(gomock.Any(), tt.bridgeConference.ID).Return(tt.bridgeConference, nil)
// 		mockReq.EXPECT().AstBridgeGet(tt.bridgeConference.AsteriskID, tt.bridgeConference.ID).Return(nil, nil)
// 		mockDB.EXPECT().BridgeGet(gomock.Any(), gomock.Any()).Return(tt.bridgeJoining, nil)
// 		mockCache.EXPECT().AsteriskAddressInternerGet(gomock.Any(), gomock.Any()).Return("", nil)
// 		mockReq.EXPECT().AstChannelCreate(tt.bridgeJoining.AsteriskID, gomock.Any(), gomock.Any(), gomock.Any(), "", "vp8", gomock.Any(), nil).Return(nil)

// 		if err := h.Join(tt.conference.ID, tt.call.ID); err != nil {
// 			t.Errorf("Wrong match. expect: ok, got: %v", err)
// 		}
// 	}
// }
