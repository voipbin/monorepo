package callhandler

import (
	"context"
	"fmt"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cfconference "monorepo/bin-conference-manager/models/conference"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"monorepo/bin-number-manager/models/number"

	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
)

func Test_GetTypeContextIncomingCall(t *testing.T) {
	type test struct {
		name       string
		domain     string
		expectType string
	}

	tests := []test{
		{
			"normal conference",
			"conference.voipbin.net",
			domainTypeConference,
		},
		{
			"pstn",
			"pstn.voipbin.net",
			domainTypePSTN,
		},
		{
			"None type",
			"randome-invalid-domain.voipbin.net",
			domainTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getDomainTypeIncomingCall(tt.domain)
			if res != tt.expectType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectType, res)
			}
		})
	}
}

func Test_Start_incoming_typeConferenceStart(t *testing.T) {

	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseUUIDCall    uuid.UUID
		responseUUIDBridge  uuid.UUID
		responseBridge      *bridge.Bridge
		responseCall        *call.Call
		responseConference  *cfconference.Conference
		responseActiveflow  *fmactiveflow.Activeflow
		responseCurTime     string

		expectBridgeName string
		expectCall       *call.Call
	}{
		{
			"normal",

			&channel.Channel{
				AsteriskID:        "80:fa:5b:5e:da:81",
				ID:                "c08ce47e-9b59-11ea-89c6-f3435f55a6ea",
				Name:              "PJSIP/in-voipbin-00000999",
				DestinationNumber: "bad943d8-9b59-11ea-b409-4ba263721f17",
				State:             ari.ChannelStateRing,
				StasisData: map[channel.StasisDataType]string{
					"context": string(channel.ContextCallIncoming),
					"domain":  "conference.voipbin.net",
				},
			},

			&commonaddress.Address{
				Type: commonaddress.TypeTel,
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "bad943d8-9b59-11ea-b409-4ba263721f17",
			},
			uuid.FromStringOrNil("666ae678-5e55-11ed-8bbd-bbd66d73cbaf"),
			uuid.FromStringOrNil("56b24806-5e56-11ed-9b77-cf2a442594d7"),
			&bridge.Bridge{
				ID: "56b24806-5e56-11ed-9b77-cf2a442594d7",
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c6914fcc-9b59-11ea-a5fc-4f4392f10a97"),
				},
				ChannelID:    "c08ce47e-9b59-11ea-89c6-f3435f55a6ea",
				Type:         call.TypeConference,
				Direction:    call.DirectionIncoming,
				FlowID:       uuid.FromStringOrNil("7d0c1efc-3fe2-11ec-b074-5b80d129f4ed"),
				ActiveflowID: uuid.FromStringOrNil("29c62b5e-a7b9-11ec-be7e-97f9236c5bb9"),
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("bad943d8-9b59-11ea-b409-4ba263721f17"),
				CustomerID: uuid.FromStringOrNil("2d6e83b0-5e56-11ed-9fcc-db15249e4a66"),
				Type:       cfconference.TypeConference,
				FlowID:     uuid.FromStringOrNil("7d0c1efc-3fe2-11ec-b074-5b80d129f4ed"),
			},
			&fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("29c62b5e-a7b9-11ec-be7e-97f9236c5bb9"),
				},
				ReferenceType: fmactiveflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c6914fcc-9b59-11ea-a5fc-4f4392f10a97"),
				FlowID:        uuid.FromStringOrNil("7d0c1efc-3fe2-11ec-b074-5b80d129f4ed"),
				CurrentAction: fmaction.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
			},
			"2020-04-18 03:22:17.995000",

			"reference_type=call,reference_id=666ae678-5e55-11ed-8bbd-bbd66d73cbaf",
			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("666ae678-5e55-11ed-8bbd-bbd66d73cbaf"),
					CustomerID: uuid.FromStringOrNil("2d6e83b0-5e56-11ed-9fcc-db15249e4a66"),
				},

				ChannelID: "c08ce47e-9b59-11ea-89c6-f3435f55a6ea",
				BridgeID:  "56b24806-5e56-11ed-9b77-cf2a442594d7",

				FlowID:       uuid.FromStringOrNil("7d0c1efc-3fe2-11ec-b074-5b80d129f4ed"),
				ActiveflowID: uuid.FromStringOrNil("29c62b5e-a7b9-11ec-be7e-97f9236c5bb9"),
				Type:         call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "bad943d8-9b59-11ea-b409-4ba263721f17",
				},
				Status: call.StatusRinging,

				Data: map[call.DataType]string{},
				Action: fmaction.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
				Direction:  call.DirectionIncoming,
				Dialroutes: []rmroute.Route{},

				TMCreate:      "2020-04-18 03:22:17.995000",
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().HangingUpWithDelay(ctx, gomock.Any(), gomock.Any(), defaultTimeoutCallDuration).Return(&channel.Channel{}, nil)

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeSIP).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestination(tt.channel, commonaddress.TypeConference).Return(tt.responseDestination)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDCall)
			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, uuid.FromStringOrNil(tt.channel.DestinationNumber)).Return(tt.responseConference, nil)

			// addCallBridge
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.responseConference.CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDBridge)
			mockBridge.EXPECT().Start(ctx, tt.channel.AsteriskID, tt.responseUUIDBridge.String(), tt.expectBridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(tt.responseBridge, nil)
			mockBridge.EXPECT().ChannelJoin(ctx, tt.responseUUIDBridge.String(), tt.channel.ID, "", false, false).Return(nil)

			mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.responseConference.CustomerID, *tt.responseSource).Return(nil, fmt.Errorf(""))

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, tt.responseConference.CustomerID, tt.responseConference.FlowID, fmactiveflow.ReferenceTypeCall, gomock.Any(), uuid.Nil).Return(tt.responseActiveflow, nil)

			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, gomock.Any()).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallCreated, tt.responseCall)
			mockReq.EXPECT().CallV1CallHealth(ctx, tt.responseCall.ID, defaultHealthDelay, 0).Return(nil)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			// action next part.
			mockDB.EXPECT().CallSetActionNextHold(ctx, gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().FlowV1ActiveflowGetNextAction(ctx, gomock.Any(), fmaction.IDStart).Return(nil, fmt.Errorf(""))

			if err := h.Start(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_StartCallHandle_IncomingTypeFlow(t *testing.T) {

	type test struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseUUIDCall    uuid.UUID
		responseUUIDBridge  uuid.UUID
		responseBridge      *bridge.Bridge
		responseNumbers     []number.Number
		responseActiveflow  *fmactiveflow.Activeflow
		responseCall        *call.Call

		expectFilters map[string]string
		expectCall    *call.Call
	}

	tests := []test{
		{
			"normal",

			&channel.Channel{
				AsteriskID:        "80:fa:5b:5e:da:81",
				ID:                "6e872d74-09ef-11eb-b3a6-37860f73cbd8",
				Name:              "PJSIP/in-voipbin-00000911",
				DestinationNumber: "+123456789",
				State:             ari.ChannelStateRing,
				StasisData: map[channel.StasisDataType]string{
					"context": string(channel.ContextCallIncoming),
					"domain":  "pstn.voipbin.net",
				},
			},

			&commonaddress.Address{
				Type: commonaddress.TypeTel,
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+123456789",
			},
			uuid.FromStringOrNil("72a902d8-09ef-11eb-92f7-1b906bde6408"),
			uuid.FromStringOrNil("ab5d36ce-5e5e-11ed-917e-a70e0240c226"),
			&bridge.Bridge{
				ID: "ab5d36ce-5e5e-11ed-917e-a70e0240c226",
			},
			[]number.Number{
				{
					ID:         uuid.FromStringOrNil("bd484f7e-09ef-11eb-9347-377b97e1b9ea"),
					CustomerID: uuid.FromStringOrNil("138ca9fa-5e5f-11ed-a85f-9f66d5e00566"),
					CallFlowID: uuid.FromStringOrNil("d2e558c2-09ef-11eb-bdec-e3ef3b78ac73"),
				},
			},
			&fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("38d55728-a7b9-11ec-9409-b77946009116"),
				},
				ReferenceType: fmactiveflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("72a902d8-09ef-11eb-92f7-1b906bde6408"),
				FlowID:        uuid.FromStringOrNil("d2e558c2-09ef-11eb-bdec-e3ef3b78ac73"),
				CurrentAction: fmaction.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72a902d8-09ef-11eb-92f7-1b906bde6408"),
				},
				ChannelID:    "6e872d74-09ef-11eb-b3a6-37860f73cbd8",
				FlowID:       uuid.FromStringOrNil("d2e558c2-09ef-11eb-bdec-e3ef3b78ac73"),
				ActiveflowID: uuid.FromStringOrNil("38d55728-a7b9-11ec-9409-b77946009116"),
				Type:         call.TypeSipService,
				Direction:    call.DirectionIncoming,
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+123456789",
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},

			map[string]string{
				"number":  "+123456789",
				"deleted": "false",
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72a902d8-09ef-11eb-92f7-1b906bde6408"),
					CustomerID: uuid.FromStringOrNil("138ca9fa-5e5f-11ed-a85f-9f66d5e00566"),
				},

				ChannelID: "6e872d74-09ef-11eb-b3a6-37860f73cbd8",
				BridgeID:  "ab5d36ce-5e5e-11ed-917e-a70e0240c226",

				FlowID:       uuid.FromStringOrNil("d2e558c2-09ef-11eb-bdec-e3ef3b78ac73"),
				ActiveflowID: uuid.FromStringOrNil("38d55728-a7b9-11ec-9409-b77946009116"),
				Type:         call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+123456789",
				},

				Status: call.StatusRinging,
				Data:   map[call.DataType]string{},
				Action: fmaction.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
				Direction:  call.DirectionIncoming,
				Dialroutes: []rmroute.Route{},

				TMCreate:      "2020-04-18T03:22:17.995000",
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().HangingUpWithDelay(ctx, tt.channel.ID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(&channel.Channel{}, nil)

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestination(tt.channel, commonaddress.TypeTel).Return(tt.responseDestination)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDCall)

			mockReq.EXPECT().NumberV1NumberGets(ctx, "", uint64(1), tt.expectFilters).Return(tt.responseNumbers, nil)
			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, uuid.Nil, tt.responseNumbers[0].CustomerID, tt.responseNumbers[0].CallFlowID, fmactiveflow.ReferenceTypeCall, gomock.Any(), uuid.Nil).Return(tt.responseActiveflow, nil)

			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.responseNumbers[0].CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDBridge)
			mockBridge.EXPECT().Start(ctx, tt.channel.AsteriskID, tt.responseUUIDBridge.String(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(tt.responseBridge, nil)
			mockBridge.EXPECT().ChannelJoin(ctx, tt.responseUUIDBridge.String(), tt.channel.ID, "", false, false).Return(nil)

			mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.responseNumbers[0].CustomerID, *tt.responseSource).Return(nil, fmt.Errorf(""))

			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.responseUUIDCall).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallCreated, tt.responseCall)
			mockReq.EXPECT().CallV1CallHealth(ctx, tt.responseCall.ID, defaultHealthDelay, 0).Return(nil)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			// action next part.
			mockDB.EXPECT().CallSetActionNextHold(ctx, gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().FlowV1ActiveflowGetNextAction(ctx, gomock.Any(), fmaction.IDStart).Return(nil, fmt.Errorf(""))

			if err := h.Start(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_StartCallHandle_Outgoing(t *testing.T) {

	tests := []struct {
		name string

		channel *channel.Channel

		responseUUIDBridge uuid.UUID
		responseBridge     *bridge.Bridge
		expectCallID       uuid.UUID
	}{
		{
			name: "normal without early exeuction",

			channel: &channel.Channel{
				AsteriskID:        "80:fa:5b:5e:da:81",
				ID:                "08959a96-8b31-11eb-a5aa-cb0965a824f8",
				Name:              "PJSIP/in-voipbin-00000912",
				DestinationNumber: "+123456789",
				State:             ari.ChannelStateRing,
				StasisData: map[channel.StasisDataType]string{
					"context": string(channel.ContextCallOutgoing),
					"domain":  "pstn.voipbin.net",
					"call_id": "086c90e2-8b31-11eb-b3a0-4ba972148103",
				},
			},

			responseUUIDBridge: uuid.FromStringOrNil("694c7770-5e60-11ed-8fe2-7f388186ee27"),
			responseBridge: &bridge.Bridge{
				ID: "694c7770-5e60-11ed-8fe2-7f388186ee27",
			},
			expectCallID: uuid.FromStringOrNil("086c90e2-8b31-11eb-b3a0-4ba972148103"),
		},
		{
			name: "normal with early exeuction",

			channel: &channel.Channel{
				AsteriskID:        "80:fa:5b:5e:da:81",
				ID:                "947f55f5-fe18-4442-9ca4-a60463ce1381",
				Name:              "PJSIP/in-voipbin-00000913",
				DestinationNumber: "+123456789",
				State:             ari.ChannelStateRing,
				StasisData: map[channel.StasisDataType]string{
					"context": string(channel.ContextCallOutgoing),
					"call_id": "d4420dd7-0b31-4bc1-b933-9c0283b8e93d",
				},
			},

			responseUUIDBridge: uuid.FromStringOrNil("694c7770-5e60-11ed-8fe2-7f388186ee27"),
			responseBridge: &bridge.Bridge{
				ID: "694c7770-5e60-11ed-8fe2-7f388186ee27",
			},
			expectCallID: uuid.FromStringOrNil("d4420dd7-0b31-4bc1-b933-9c0283b8e93d"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()
			mockChannel.EXPECT().HangingUpWithDelay(ctx, tt.channel.ID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration).Return(&channel.Channel{}, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDBridge)
			mockBridge.EXPECT().Start(ctx, tt.channel.AsteriskID, tt.responseUUIDBridge.String(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(tt.responseBridge, nil)
			mockBridge.EXPECT().ChannelJoin(ctx, tt.responseUUIDBridge.String(), tt.channel.ID, "", false, false).Return(nil)
			mockDB.EXPECT().CallSetBridgeID(ctx, tt.expectCallID, gomock.Any()).Return(nil)
			mockChannel.EXPECT().Dial(ctx, tt.channel.ID, tt.channel.ID, defaultDialTimeout).Return(nil)

			if err := h.Start(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_StartHandlerContextExternalMedia(t *testing.T) {

	tests := []struct {
		name string

		channel *channel.Channel

		responseExternalMedia *externalmedia.ExternalMedia
		responseCall          *call.Call

		expectExternalMediaID uuid.UUID
		expectBridgeID        string
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				AsteriskID:        "80:fa:5b:5e:da:81",
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				Name:              "PJSIP/in-voipbin-00000915",
				DestinationNumber: "+123456789",
				State:             ari.ChannelStateRing,
				StasisData: map[channel.StasisDataType]string{
					"context":           string(channel.ContextExternalMedia),
					"external_media_id": "45efbb3c-b33d-11ef-8648-fbef93b5f7dc",
				},
			},

			responseExternalMedia: &externalmedia.ExternalMedia{
				ID:            uuid.FromStringOrNil("45efbb3c-b33d-11ef-8648-fbef93b5f7dc"),
				ReferenceType: externalmedia.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("0648d6c0-0301-11ec-818e-53865044b15c"),
			},
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0648d6c0-0301-11ec-818e-53865044b15c"),
				},
				BridgeID: "6acf04f2-b33e-11ef-b32f-8f571d44cc7a",
			},

			expectExternalMediaID: uuid.FromStringOrNil("45efbb3c-b33d-11ef-8648-fbef93b5f7dc"),
			expectBridgeID:        "6acf04f2-b33e-11ef-b32f-8f571d44cc7a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &callHandler{
				reqHandler:           mockReq,
				notifyHandler:        mockNotify,
				db:                   mockDB,
				channelHandler:       mockChannel,
				bridgeHandler:        mockBridge,
				externalMediaHandler: mockExternal,
			}
			ctx := context.Background()

			mockExternal.EXPECT().Get(ctx, tt.expectExternalMediaID).Return(tt.responseExternalMedia, nil)
			mockDB.EXPECT().CallGet(ctx, tt.responseExternalMedia.ReferenceID).Return(tt.responseCall, nil)

			mockBridge.EXPECT().ChannelJoin(ctx, tt.expectBridgeID, tt.channel.ID, "", false, false).Return(nil)
			if err := h.Start(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_StartHandlerContextExternalSnoop(t *testing.T) {

	tests := []struct {
		name string

		channel        *channel.Channel
		expectBridgeID string
	}{
		{
			"normal",

			&channel.Channel{
				ID:         "asterisk-call-5765d977d8-c4k5q-1629607067.6639",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "Snoop/asterisk-call-5765d977d8-c4k5q-1629250154.132-00000000",
				StasisData: map[channel.StasisDataType]string{
					"context":   string(channel.ContextExternalSoop),
					"bridge_id": "d6aecd56-0301-11ec-aee0-77d9356147eb",
					"call_id":   "da646758-0301-11ec-b3eb-f3c05485b756",
				},
			},
			"d6aecd56-0301-11ec-aee0-77d9356147eb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockBridge.EXPECT().ChannelJoin(ctx, tt.expectBridgeID, tt.channel.ID, "", false, false).Return(nil)
			if err := h.Start(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Start_ContextJoinCall(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel

		expectBridgeID string
	}{
		{
			"normal",
			&channel.Channel{
				ID:         "asterisk-call-06627464-431a-11ec-bda3-2f0d6128b98f",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "Snoop/asterisk-call-5765d977d8-c4k5q-1629250154.139-00000000",
				StasisData: map[channel.StasisDataType]string{
					"context":       string(channel.ContextJoinCall),
					"bridge_id":     "ed08cbf8-4319-11ec-a768-23af5da287d4",
					"call_id":       "ed4ba266-4319-11ec-80b7-9f3d3acb4aa0",
					"confbridge_id": "ed70a890-4319-11ec-a8dc-8baa3bdb6a39",
				},
			},

			"ed08cbf8-4319-11ec-a768-23af5da287d4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockBridge.EXPECT().ChannelJoin(ctx, tt.expectBridgeID, tt.channel.ID, "", false, false).Return(nil)
			mockChannel.EXPECT().Dial(ctx, tt.channel.ID, "", defaultDialTimeout).Return(nil)

			if err := h.Start(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_getAddressOwner(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		address    *commonaddress.Address

		responseAgent *amagent.Agent

		expectAgentID      uuid.UUID
		expectResOwnerType commonidentity.OwnerType
		expectResOwnerID   uuid.UUID
	}{
		{
			name: "normal - address type is agent",

			customerID: uuid.FromStringOrNil("9129ad1a-2fd5-11ef-af80-1f74bf8dbf2b"),
			address: &commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "91a56bc6-2fd5-11ef-b664-bf13af632f01",
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91a56bc6-2fd5-11ef-b664-bf13af632f01"),
					CustomerID: uuid.FromStringOrNil("9129ad1a-2fd5-11ef-af80-1f74bf8dbf2b"),
				},
			},

			expectAgentID:      uuid.FromStringOrNil("91a56bc6-2fd5-11ef-b664-bf13af632f01"),
			expectResOwnerType: commonidentity.OwnerTypeAgent,
			expectResOwnerID:   uuid.FromStringOrNil("91a56bc6-2fd5-11ef-b664-bf13af632f01"),
		},
		{
			name: "normal - address type is not agent",

			customerID: uuid.FromStringOrNil("9129ad1a-2fd5-11ef-af80-1f74bf8dbf2b"),
			address: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+123456789",
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91e8e824-2fd5-11ef-a8d1-6f57a27e6c6f"),
					CustomerID: uuid.FromStringOrNil("9129ad1a-2fd5-11ef-af80-1f74bf8dbf2b"),
				},
			},

			expectResOwnerType: commonidentity.OwnerTypeAgent,
			expectResOwnerID:   uuid.FromStringOrNil("91e8e824-2fd5-11ef-a8d1-6f57a27e6c6f"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			if tt.expectAgentID != uuid.Nil {
				mockReq.EXPECT().AgentV1AgentGet(ctx, tt.expectAgentID).Return(tt.responseAgent, nil)
			} else {
				mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.customerID, *tt.address).Return(tt.responseAgent, nil)
			}

			resOwnerType, resOwnerID, err := h.getAddressOwner(ctx, tt.customerID, tt.address)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if resOwnerType != tt.expectResOwnerType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectResOwnerType, resOwnerType)
			}
			if resOwnerID != tt.expectResOwnerID {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectResOwnerID, resOwnerID)
			}
		})
	}
}
