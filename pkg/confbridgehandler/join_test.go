package confbridgehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestCreateEndpointTarget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := confbridgeHandler{
		reqHandler: mockReq,
		db:         mockDB,
		cache:      mockCache,
	}

	type test struct {
		name            string
		asteriskAddress string
		confbridge      *confbridge.Confbridge
		bridge          *bridge.Bridge
		expectEndpoint  string
	}

	tests := []test{
		{
			"normal",
			"10.10.10.10",
			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("a2a8a798-36e1-11ec-a28e-df4baed10101"),
				BridgeID: "a2e97408-36e1-11ec-8f9e-77e7152bdb1c",
			},
			&bridge.Bridge{
				ID:         "a2e97408-36e1-11ec-8f9e-77e7152bdb1c",
				AsteriskID: "00:11:22:33:44:55",
			},
			"PJSIP/conf-join/sip:a2e97408-36e1-11ec-8f9e-77e7152bdb1c@10.10.10.10:5060",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.confbridge.BridgeID).Return(tt.bridge, nil)
			mockCache.EXPECT().AsteriskAddressInternalGet(gomock.Any(), tt.bridge.AsteriskID).Return(tt.asteriskAddress, nil)

			res, err := h.createEndpointTarget(context.Background(), tt.confbridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectEndpoint {
				t.Errorf("Wrong match. expect: %s\ngot:%s\n", tt.expectEndpoint, res)
			}
		})
	}
}

func TestJoin(t *testing.T) {
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

	type test struct {
		name       string
		confbridge *confbridge.Confbridge
		call       *call.Call

		bridge *bridge.Bridge
	}

	tests := []test{
		{
			"has bridge id",
			&confbridge.Confbridge{
				ID:           uuid.FromStringOrNil("9c637510-36e2-11ec-b37c-63ed644a2629"),
				BridgeID:     "a5c525ec-dca0-11ea-b139-17780451d9da",
				ConferenceID: uuid.FromStringOrNil("e5a73f1c-36e3-11ec-afc3-330513db8c19"),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("00b8301e-9f1d-11ea-a08e-038ccf4318cd"),
				AsteriskID: "00:11:22:33:44:55",
				ChannelID:  "15feb3f8-9f1d-11ea-a707-7f644f1ae186",
				Type:       call.TypeConference,
			},
			&bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "a5c525ec-dca0-11ea-b139-17780451d9da",
				TMDelete:   defaultTimeStamp,
			},
		},
		{
			"has no bridge id",
			&confbridge.Confbridge{
				ID:           uuid.FromStringOrNil("20fb7f68-38e4-11ec-a269-8f5f84d4c603"),
				ConferenceID: uuid.FromStringOrNil("211674b2-38e4-11ec-ac6d-f3a275453b52"),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("2133ea06-38e4-11ec-a400-df96309626a9"),
				AsteriskID: "00:11:22:33:44:55",
				ChannelID:  "15feb3f8-9f1d-11ea-a707-7f644f1ae186",
				Type:       call.TypeConference,
			},
			&bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "214c8606-38e4-11ec-8960-0fff1696a6b1",
				TMDelete:   defaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(gomock.Any(), tt.confbridge.ID).Return(tt.confbridge, nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)

			mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
			if tt.confbridge.BridgeID != "" {
				mockDB.EXPECT().BridgeGet(gomock.Any(), tt.confbridge.BridgeID).Return(tt.bridge, nil)
				mockReq.EXPECT().AstBridgeGet(tt.bridge.AsteriskID, tt.bridge.ID).Return(tt.bridge, nil)
			} else {
				// todo: check bridge creation
				mockReq.EXPECT().AstBridgeCreate(requesthandler.AsteriskIDConference, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing}).Return(nil)
				mockDB.EXPECT().BridgeGetUntilTimeout(gomock.Any(), gomock.Any()).Return(tt.bridge, nil)
				mockDB.EXPECT().ConfbridgeSetBridgeID(gomock.Any(), gomock.Any(), tt.bridge.ID).Return(nil)
				mockDB.EXPECT().ConfbridgeGet(gomock.Any(), tt.confbridge.ID).Return(tt.confbridge, nil)
			}
			mockDB.EXPECT().BridgeGet(gomock.Any(), gomock.Any()).Return(tt.bridge, nil)
			mockCache.EXPECT().AsteriskAddressInternalGet(gomock.Any(), tt.bridge.AsteriskID).Return("test.com", nil)

			mockReq.EXPECT().AstChannelCreate(tt.call.AsteriskID, gomock.Any(), gomock.Any(), gomock.Any(), "", "vp8", "", nil).Return(nil)

			if err := h.Join(ctx, tt.confbridge.ID, tt.call.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
