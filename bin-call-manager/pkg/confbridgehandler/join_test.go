package confbridgehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Join(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		callID uuid.UUID

		responseBridge                  *bridge.Bridge
		responseAsteriskAddressInternal string
		rsponseCall                     *call.Call

		responseConfbridge  *confbridge.Confbridge
		responseUUIDBridge  uuid.UUID
		responseUUIDChannel uuid.UUID

		expectBridgeArgs             string
		expectBridgeTypes            []bridge.Type
		expectChannelArgs            string
		expectChannelDialDestination string
		expectReqVariables           map[string]string
	}{
		{
			name: "has valid bridge id",

			id:     uuid.FromStringOrNil("9c637510-36e2-11ec-b37c-63ed644a2629"),
			callID: uuid.FromStringOrNil("00b8301e-9f1d-11ea-a08e-038ccf4318cd"),

			responseConfbridge: &confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("9c637510-36e2-11ec-b37c-63ed644a2629"),
				BridgeID: "a5c525ec-dca0-11ea-b139-17780451d9da",
			},

			rsponseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("00b8301e-9f1d-11ea-a08e-038ccf4318cd"),
				},
				ChannelID: "15feb3f8-9f1d-11ea-a707-7f644f1ae186",
				BridgeID:  "7154276e-a3cb-11ed-bb8d-f307ed271462",
				Type:      call.TypeConference,
			},
			responseBridge: &bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "a5c525ec-dca0-11ea-b139-17780451d9da",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseAsteriskAddressInternal: "test.com",
			responseUUIDChannel:             uuid.FromStringOrNil("2b6f7cb4-9c76-11ed-9714-8b312d1ce8a5"),

			expectChannelArgs:            "context_type=call,context=call-join,confbridge_id=9c637510-36e2-11ec-b37c-63ed644a2629,bridge_id=7154276e-a3cb-11ed-bb8d-f307ed271462,call_id=00b8301e-9f1d-11ea-a08e-038ccf4318cd",
			expectChannelDialDestination: "PJSIP/conf-join/sip:a5c525ec-dca0-11ea-b139-17780451d9da@test.com:5060",
			expectReqVariables: map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderCallID + ")":       "00b8301e-9f1d-11ea-a08e-038ccf4318cd",
				"PJSIP_HEADER(add," + common.SIPHeaderConfbridgeID + ")": "9c637510-36e2-11ec-b37c-63ed644a2629",
			},
		},
		{
			name: "has no valid bridge id",

			id:     uuid.FromStringOrNil("20fb7f68-38e4-11ec-a269-8f5f84d4c603"),
			callID: uuid.FromStringOrNil("2133ea06-38e4-11ec-a400-df96309626a9"),

			responseConfbridge: &confbridge.Confbridge{
				ID:   uuid.FromStringOrNil("20fb7f68-38e4-11ec-a269-8f5f84d4c603"),
				Type: confbridge.TypeConnect,
			},
			rsponseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2133ea06-38e4-11ec-a400-df96309626a9"),
				},
				ChannelID: "15feb3f8-9f1d-11ea-a707-7f644f1ae186",
				BridgeID:  "5909845a-a3cc-11ed-a8ec-cf8e4729c180",
				Type:      call.TypeFlow,
			},
			responseBridge: &bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "214c8606-38e4-11ec-8960-0fff1696a6b1",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseAsteriskAddressInternal: "test.com",

			responseUUIDBridge:  uuid.FromStringOrNil("214c8606-38e4-11ec-8960-0fff1696a6b1"),
			responseUUIDChannel: uuid.FromStringOrNil("2d20d1ac-9c76-11ed-8d30-9bd30a670223"),

			expectBridgeArgs:             "reference_type=confbridge,reference_id=20fb7f68-38e4-11ec-a269-8f5f84d4c603",
			expectBridgeTypes:            []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia},
			expectChannelArgs:            "context_type=call,context=call-join,confbridge_id=20fb7f68-38e4-11ec-a269-8f5f84d4c603,bridge_id=5909845a-a3cc-11ed-a8ec-cf8e4729c180,call_id=2133ea06-38e4-11ec-a400-df96309626a9",
			expectChannelDialDestination: "PJSIP/conf-join/sip:214c8606-38e4-11ec-8960-0fff1696a6b1@test.com:5060",
			expectReqVariables: map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderCallID + ")":       "2133ea06-38e4-11ec-a400-df96309626a9",
				"PJSIP_HEADER(add," + common.SIPHeaderConfbridgeID + ")": "20fb7f68-38e4-11ec-a269-8f5f84d4c603",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := confbridgeHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				cache:          mockCache,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			mockReq.EXPECT().CallV1CallGet(ctx, tt.callID).Return(tt.rsponseCall, nil)

			if tt.responseConfbridge.BridgeID != "" {
				mockBridge.EXPECT().IsExist(ctx, tt.responseConfbridge.BridgeID).Return(true)
			} else {
				mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridge, nil)
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDBridge)
				mockBridge.EXPECT().Start(ctx, requesthandler.AsteriskIDConference, tt.responseUUIDBridge.String(), tt.expectBridgeArgs, tt.expectBridgeTypes).Return(tt.responseBridge, nil)

				mockDB.EXPECT().ConfbridgeSetBridgeID(ctx, tt.id, tt.responseBridge.ID).Return(nil)
				mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridge, nil)
			}
			mockBridge.EXPECT().Get(ctx, tt.responseConfbridge.BridgeID).Return(tt.responseBridge, nil)
			mockCache.EXPECT().AsteriskAddressInternalGet(ctx, tt.responseBridge.AsteriskID).Return(tt.responseAsteriskAddressInternal, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)
			mockChannel.EXPECT().StartChannelWithBaseChannel(ctx, tt.rsponseCall.ChannelID, tt.responseUUIDChannel.String(), tt.expectChannelArgs, tt.expectChannelDialDestination, "", "vp8", "", tt.expectReqVariables).Return(&channel.Channel{}, nil)

			if err := h.Join(ctx, tt.id, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_createConfbridgeBridge(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConfbridge *confbridge.Confbridge
		responseUUID       uuid.UUID
		responseBridge     *bridge.Bridge

		expectBridgeName  string
		expectBridgeTypes []bridge.Type
	}{
		{
			name: "type connect",

			id: uuid.FromStringOrNil("b385b994-a3cc-11ed-a0af-e74c91fe4e29"),

			responseConfbridge: &confbridge.Confbridge{
				ID:   uuid.FromStringOrNil("b385b994-a3cc-11ed-a0af-e74c91fe4e29"),
				Type: confbridge.TypeConnect,
			},

			responseUUID: uuid.FromStringOrNil("b3b31362-a3cc-11ed-9e1c-c390c233119e"),
			responseBridge: &bridge.Bridge{
				ID: "b3b31362-a3cc-11ed-9e1c-c390c233119e",
			},

			expectBridgeName:  "reference_type=confbridge,reference_id=b385b994-a3cc-11ed-a0af-e74c91fe4e29",
			expectBridgeTypes: []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia},
		},
		{
			name: "type conference",

			id: uuid.FromStringOrNil("248d449e-a3ce-11ed-a551-cf943ca821db"),

			responseConfbridge: &confbridge.Confbridge{
				ID:   uuid.FromStringOrNil("248d449e-a3ce-11ed-a551-cf943ca821db"),
				Type: confbridge.TypeConference,
			},

			responseUUID: uuid.FromStringOrNil("24b4ffca-a3ce-11ed-9e6e-17ed8f86b8f9"),
			responseBridge: &bridge.Bridge{
				ID: "24b4ffca-a3ce-11ed-9e6e-17ed8f86b8f9",
			},

			expectBridgeName:  "reference_type=confbridge,reference_id=248d449e-a3ce-11ed-a551-cf943ca821db",
			expectBridgeTypes: []bridge.Type{bridge.TypeMixing},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := confbridgeHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				bridgeHandler: mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockBridge.EXPECT().Start(ctx, requesthandler.AsteriskIDConference, tt.responseUUID.String(), tt.expectBridgeName, tt.expectBridgeTypes).Return(tt.responseBridge, nil)
			mockDB.EXPECT().ConfbridgeSetBridgeID(ctx, tt.responseConfbridge.ID, tt.responseBridge.ID).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseConfbridge.ID).Return(tt.responseConfbridge, nil)

			res, err := h.createConfbridgeBridge(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConfbridge, res) {
				t.Errorf("Wrong match. expect: %s\ngot:%s\n", tt.responseConfbridge, res)
			}
		})
	}
}

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
