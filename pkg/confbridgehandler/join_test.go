package confbridgehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_createEndpointTarget(t *testing.T) {

	tests := []struct {
		name            string
		asteriskAddress string
		confbridge      *confbridge.Confbridge
		bridge          *bridge.Bridge
		expectEndpoint  string
	}{
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := confbridgeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				bridgeHandler: mockBridge,
			}

			ctx := context.Background()

			mockBridge.EXPECT().Get(ctx, tt.confbridge.BridgeID).Return(tt.bridge, nil)
			mockCache.EXPECT().AsteriskAddressInternalGet(ctx, tt.bridge.AsteriskID).Return(tt.asteriskAddress, nil)

			res, err := h.createEndpointTarget(ctx, tt.confbridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectEndpoint {
				t.Errorf("Wrong match. expect: %s\ngot:%s\n", tt.expectEndpoint, res)
			}
		})
	}
}

func Test_Join(t *testing.T) {

	tests := []struct {
		name       string
		confbridge *confbridge.Confbridge
		call       *call.Call

		bridge *bridge.Bridge

		expectReqVariables map[string]string
	}{
		{
			"has bridge id",
			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("9c637510-36e2-11ec-b37c-63ed644a2629"),
				BridgeID: "a5c525ec-dca0-11ea-b139-17780451d9da",
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
				TMDelete:   dbhandler.DefaultTimeStamp,
			},

			map[string]string{
				"PJSIP_HEADER(add,VB-CALL-ID)":       "00b8301e-9f1d-11ea-a08e-038ccf4318cd",
				"PJSIP_HEADER(add,VB-CONFBRIDGE-ID)": "9c637510-36e2-11ec-b37c-63ed644a2629",
			},
		},
		{
			"has no bridge id",
			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("20fb7f68-38e4-11ec-a269-8f5f84d4c603"),
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
				TMDelete:   dbhandler.DefaultTimeStamp,
			},

			map[string]string{
				"PJSIP_HEADER(add,VB-CALL-ID)":       "2133ea06-38e4-11ec-a400-df96309626a9",
				"PJSIP_HEADER(add,VB-CONFBRIDGE-ID)": "20fb7f68-38e4-11ec-a269-8f5f84d4c603",
			},
		},
		{
			"call has answered",
			&confbridge.Confbridge{
				ID: uuid.FromStringOrNil("670fc934-972e-11ec-a07d-d3be053b11a3"),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("67368c4a-972e-11ec-83e0-570b3a259d76"),
				AsteriskID: "00:11:22:33:44:55",
				ChannelID:  "675b348c-972e-11ec-a069-b31fae066d64",
				Type:       call.TypeConference,
				Status:     call.StatusProgressing,
			},
			&bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "732f7480-972e-11ec-bce4-0f6c7a174b02",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},

			map[string]string{
				"PJSIP_HEADER(add,VB-CALL-ID)":       "67368c4a-972e-11ec-83e0-570b3a259d76",
				"PJSIP_HEADER(add,VB-CONFBRIDGE-ID)": "670fc934-972e-11ec-a07d-d3be053b11a3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := confbridgeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
				bridgeHandler: mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID).Return(tt.confbridge, nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)

			if tt.call.Status == call.StatusRinging {
				mockReq.EXPECT().AstChannelAnswer(ctx, tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
			}
			if tt.confbridge.BridgeID != "" {
				mockBridge.EXPECT().Get(ctx, tt.confbridge.BridgeID).Return(tt.bridge, nil)
				mockReq.EXPECT().AstBridgeGet(ctx, tt.bridge.AsteriskID, tt.bridge.ID).Return(tt.bridge, nil)
			} else {
				// todo: check bridge creation
				mockReq.EXPECT().AstBridgeCreate(ctx, requesthandler.AsteriskIDConference, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing}).Return(nil)
				mockBridge.EXPECT().GetWithTimeout(ctx, gomock.Any(), gomock.Any()).Return(tt.bridge, nil)
				mockDB.EXPECT().ConfbridgeSetBridgeID(ctx, gomock.Any(), tt.bridge.ID).Return(nil)
				mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID).Return(tt.confbridge, nil)
			}
			mockBridge.EXPECT().Get(ctx, tt.confbridge.BridgeID).Return(tt.bridge, nil)
			mockCache.EXPECT().AsteriskAddressInternalGet(ctx, tt.bridge.AsteriskID).Return("test.com", nil)

			mockReq.EXPECT().AstChannelCreate(ctx, tt.call.AsteriskID, gomock.Any(), gomock.Any(), gomock.Any(), "", "vp8", "", tt.expectReqVariables).Return(&channel.Channel{}, nil)

			if err := h.Join(ctx, tt.confbridge.ID, tt.call.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
