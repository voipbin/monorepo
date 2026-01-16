package callhandler

import (
	"context"
	"reflect"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	rmprovider "monorepo/bin-route-manager/models/provider"
	rmroute "monorepo/bin-route-manager/models/route"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

func Test_CreateCallOutgoing_TypeSIP(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		flowID         uuid.UUID
		activeflowID   uuid.UUID
		masterCallID   uuid.UUID
		source         commonaddress.Address
		destination    commonaddress.Address
		earlyExecution bool
		connect        bool

		responseActiveflow  *fmactiveflow.Activeflow
		responseAgent       *amagent.Agent
		responseUUIDChannel uuid.UUID

		expectDialrouteTarget string
		expectCall            *call.Call
		expectArgs            string
		expectEndpointDst     string
		expectVariables       map[string]string
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
			customerID:   uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
			flowID:       uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
			activeflowID: uuid.FromStringOrNil("679f0eb2-8c21-41a6-876d-9d778b1b0167"),
			masterCallID: uuid.FromStringOrNil("5935ff8a-8c8f-11ec-b26a-3fee169eaf45"),
			source: commonaddress.Address{
				Type:       commonaddress.TypeSIP,
				Target:     "testsrc@test.com",
				TargetName: "test",
			},
			destination: commonaddress.Address{
				Type:       commonaddress.TypeSIP,
				Target:     "testoutgoing@test.com",
				TargetName: "test target",
			},
			earlyExecution: true,
			connect:        true,

			responseActiveflow: &fmactiveflow.Activeflow{
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1aa075dc-2bfe-11ef-9203-37278cb94d16"),
					CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				},
			},
			responseUUIDChannel: uuid.FromStringOrNil("80d67b3a-5f3b-11ed-a709-0f2943ef0184"),

			expectDialrouteTarget: "",
			expectCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
					CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("1aa075dc-2bfe-11ef-9203-37278cb94d16"),
				},
				ChannelID: "80d67b3a-5f3b-11ed-a709-0f2943ef0184",
				FlowID:    uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
				Type:      call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Status:      call.StatusDialing,
				Direction:   call.DirectionOutgoing,
				GroupcallID: uuid.Nil,
				Source: commonaddress.Address{
					Type:       commonaddress.TypeSIP,
					Target:     "testsrc@test.com",
					TargetName: "test",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeSIP,
					Target:     "testoutgoing@test.com",
					TargetName: "test target",
				},
				Data: map[call.DataType]string{
					call.DataTypeEarlyExecution:            "true",
					call.DataTypeExecuteNextMasterOnHangup: "true",
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},

				Dialroutes: []rmroute.Route{},

				TMCreate:      "2021-02-19 06:32:14.621",
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},
			expectArgs:        "context_type=call,context=call-out,call_id=f1afa9ce-ecb2-11ea-ab94-a768ab787da0,transport=udp,direction=outgoing",
			expectEndpointDst: "pjsip/call-out/sip:testoutgoing@test.com",
			expectVariables: map[string]string{
				"CALLERID(name)": "test",
				"CALLERID(num)":  "testsrc@test.com",
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
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

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.customerID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id, uuid.Nil).Return(tt.responseActiveflow, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.customerID, tt.destination).Return(tt.responseAgent, nil)
			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)
			mockReq.EXPECT().CallV1CallHealth(ctx, tt.expectCall.ID, defaultHealthDelay, 0).Return(nil)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			if tt.masterCallID != uuid.Nil {
				mockDB.EXPECT().CallTXStart(tt.masterCallID).Return(nil, &call.Call{}, nil)
				mockDB.EXPECT().CallTXAddChainedCallID(gomock.Any(), tt.masterCallID, tt.expectCall.ID).Return(nil)
				mockDB.EXPECT().CallSetMasterCallID(ctx, tt.expectCall.ID, tt.masterCallID).Return(nil)
				mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

				mockDB.EXPECT().CallGet(ctx, tt.masterCallID).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

				mockDB.EXPECT().CallGet(ctx, tt.expectCall.ID).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())
			}

			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, gomock.Any(), tt.expectArgs, tt.expectEndpointDst, "", "", "", tt.expectVariables).Return(&channel.Channel{}, nil)

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, tt.earlyExecution, tt.connect)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CreateCallOutgoing_TypeTel(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		flowID         uuid.UUID
		activeflowID   uuid.UUID
		masterCallID   uuid.UUID
		source         commonaddress.Address
		destination    commonaddress.Address
		earlyExecution bool
		connect        bool

		responseActiveflow  *fmactiveflow.Activeflow
		responseRoutes      []rmroute.Route
		responseAgent       *amagent.Agent
		responseUUIDChannel uuid.UUID
		responseProvider    *rmprovider.Provider

		expectDialrouteTarget string
		expectCall            *call.Call
		expectProviderID      uuid.UUID
		expectArgs            string
		expectEndpointDst     string
		expectVariables       map[string]string
	}{
		{
			name: "have all",

			id:           uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
			customerID:   uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
			flowID:       uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
			activeflowID: uuid.FromStringOrNil("11e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
			masterCallID: uuid.FromStringOrNil("61c0fe66-8c8f-11ec-873a-ff90a846a02f"),
			source: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+99999888",
				TargetName: "test",
			},
			destination: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821121656521",
				TargetName: "test target",
			},
			earlyExecution: true,
			connect:        true,

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
				},
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
			responseRoutes: []rmroute.Route{
				{
					ID:         uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
					ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
				},
			},
			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1b095188-2bfe-11ef-a746-7f4de3b06e46"),
				},
			},
			responseUUIDChannel: uuid.FromStringOrNil("d948969e-5de3-11ed-94f5-137ec429b6b6"),
			responseProvider: &rmprovider.Provider{
				Hostname: "sip.telnyx.com",
			},

			expectDialrouteTarget: "+82",
			expectCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
					CustomerID: uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("1b095188-2bfe-11ef-a746-7f4de3b06e46"),
				},
				ChannelID:      "d948969e-5de3-11ed-94f5-137ec429b6b6",
				FlowID:         uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
				ActiveflowID:   uuid.FromStringOrNil("11e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Status:         call.StatusDialing,
				Data: map[call.DataType]string{
					call.DataTypeEarlyExecution:            "true",
					call.DataTypeExecuteNextMasterOnHangup: "true",
				},
				Direction:   call.DirectionOutgoing,
				GroupcallID: uuid.Nil,
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+99999888",
					TargetName: "test",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821121656521",
					TargetName: "test target",
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},
				DialrouteID: uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
						ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
					},
				},

				TMCreate:      "2021-02-19 06:32:14.621",
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},
			expectProviderID:  uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
			expectArgs:        "context_type=call,context=call-out,call_id=b7c40962-07fb-11eb-bb82-a3bd16bf1bd9,transport=udp,direction=outgoing",
			expectEndpointDst: "pjsip/call-out/sip:+821121656521@sip.telnyx.com;transport=udp",
			expectVariables: map[string]string{
				"CALLERID(name)": "test",
				"CALLERID(num)":  "+99999888",
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
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

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.customerID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id, uuid.Nil).Return(tt.responseActiveflow, nil)
			// getDialURI
			mockReq.EXPECT().RouteV1DialrouteList(ctx, gomock.Any()).Return(tt.responseRoutes, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)

			mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.customerID, tt.destination).Return(tt.responseAgent, nil)

			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)
			mockReq.EXPECT().CallV1CallHealth(ctx, tt.expectCall.ID, defaultHealthDelay, 0).Return(nil)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			if tt.masterCallID != uuid.Nil {
				mockDB.EXPECT().CallTXStart(tt.masterCallID).Return(nil, &call.Call{}, nil)
				mockDB.EXPECT().CallTXAddChainedCallID(gomock.Any(), tt.masterCallID, tt.expectCall.ID).Return(nil)
				mockDB.EXPECT().CallSetMasterCallID(ctx, tt.expectCall.ID, tt.masterCallID).Return(nil)
				mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

				mockDB.EXPECT().CallGet(ctx, tt.masterCallID).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

				mockDB.EXPECT().CallGet(ctx, tt.expectCall.ID).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())
			}

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)

			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, gomock.Any(), tt.expectArgs, tt.expectEndpointDst, "", "", "", tt.expectVariables).Return(&channel.Channel{}, nil)

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, tt.earlyExecution, tt.connect)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_createCallsOutgoingGroupcall_endpoint(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		flowID       uuid.UUID
		masterCallID uuid.UUID
		source       *commonaddress.Address
		destination  *commonaddress.Address

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			flowID:       uuid.FromStringOrNil("e9ebb18c-b5c4-11ed-9775-cf1b5f3ac127"),
			masterCallID: uuid.FromStringOrNil("7ca3f4f7-a5c3-4df3-8a8a-a008ac2380be"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "test-exten",
			},

			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a14f3af-68cd-4b64-b6db-856cb15cc1cc"),
				},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("9da13d85-1a50-46d7-a4f6-e1a70650a648"),
					uuid.FromStringOrNil("8815fce3-7560-4574-a659-20905f928f5a"),
				},
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				notifyHandler:    mockNotify,
				channelHandler:   mockChannel,
				groupcallHandler: mockGroupcall,
			}
			ctx := context.Background()

			mockGroupcall.EXPECT().Start(ctx, uuid.Nil, tt.customerID, tt.flowID, tt.source, []commonaddress.Address{*tt.destination}, tt.masterCallID, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers).Return(tt.responseGroupcall, nil)

			res, err := h.createCallsOutgoingGroupcall(ctx, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_createCallsOutgoingGroupcall_agent(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		flowID       uuid.UUID
		masterCallID uuid.UUID
		source       *commonaddress.Address
		destination  *commonaddress.Address

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "agent ring method ring linear",

			customerID:   uuid.FromStringOrNil("f5979302-e274-11ed-8e02-f7e891b8718e"),
			flowID:       uuid.FromStringOrNil("f5cfdd2a-e274-11ed-a8cb-eb89c814da27"),
			masterCallID: uuid.FromStringOrNil("f5fc5972-e274-11ed-999b-2743d4d4b02a"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "8794f182-e275-11ed-b53f-93e62e46ec3d",
			},

			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f6223d68-e274-11ed-805f-4329b5f9076e"),
				},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("f64b0bd0-e274-11ed-8457-63d2b832a8c0"),
					uuid.FromStringOrNil("f673400a-e274-11ed-920b-535097d30c6f"),
				},
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				notifyHandler:    mockNotify,
				channelHandler:   mockChannel,
				groupcallHandler: mockGroupcall,
			}
			ctx := context.Background()

			mockGroupcall.EXPECT().Start(ctx, uuid.Nil, tt.customerID, tt.flowID, tt.source, []commonaddress.Address{*tt.destination}, tt.masterCallID, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers).Return(tt.responseGroupcall, nil)

			res, err := h.createCallsOutgoingGroupcall(ctx, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_getDialURI_Tel(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseProvider *rmprovider.Provider

		expectProviderID uuid.UUID
		expectTarget     string
		expectRes        string
	}{
		{
			"normal",

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("04e5d530-5d96-11ed-bbc8-cfb95f6d6085"),
					CustomerID: uuid.FromStringOrNil("f7a14b8c-534c-11ed-9fb1-c7c376f2730b"),
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821121656521",
				},
				DialrouteID: uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
						ProviderID: uuid.FromStringOrNil("8730a3da-5350-11ed-aa47-7f44741127c1"),
					},
				},
			},

			&rmprovider.Provider{
				ID:       uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
				Hostname: "sip.telnyx.com",
			},

			uuid.FromStringOrNil("8730a3da-5350-11ed-aa47-7f44741127c1"),
			"+82",
			"pjsip/call-out/sip:+821121656521@sip.telnyx.com;transport=udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)

			res, err := h.getDialURI(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_getDialURI_SIP(t *testing.T) {

	tests := []struct {
		name string

		c *call.Call

		expectRes string
	}{
		{
			"normal",

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3c28f5fe-5d96-11ed-bf69-9340492cc88d"),
					CustomerID: uuid.FromStringOrNil("22139104-534d-11ed-aba9-e73d8b8e1c43"),
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "test@test.com",
				},
			},

			"pjsip/call-out/sip:test@test.com",
		},
		{
			"normal",

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5e0329d8-5d96-11ed-a009-2763e323daa8"),
					CustomerID: uuid.FromStringOrNil("dfd086b2-534c-11ed-b905-93a3b56e1ae8"),
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "test@test.com",
				},
			},

			"pjsip/call-out/sip:test@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			res, err := h.getDialURI(ctx, tt.c)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_getDialURI_error(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call
	}{
		{
			"supported address type agent",

			&call.Call{
				Destination: commonaddress.Address{
					Type: commonaddress.TypeAgent,
				},
			},
		},
		{
			"supported address type conference",

			&call.Call{
				Destination: commonaddress.Address{
					Type: commonaddress.TypeConference,
				},
			},
		},
		{
			"supported address type extension",

			&call.Call{
				Destination: commonaddress.Address{
					Type: commonaddress.TypeExtension,
				},
			},
		},
		{
			"supported address type line",

			&call.Call{
				Destination: commonaddress.Address{
					Type: commonaddress.TypeLine,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			_, err := h.getDialURI(ctx, tt.call)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func Test_createChannel(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseProvider *rmprovider.Provider

		expectProviderID uuid.UUID
		expectArgs       string
		expectDialURI    string
		expectVariables  map[string]string
	}{
		{
			name: "normal",

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7e0a846a-5d96-11ed-9005-07794a4f93cb"),
					CustomerID: uuid.FromStringOrNil("6f3fd136-534d-11ed-90a2-ff71219800e5"),
				},

				ChannelID: "7fd0947a-5f39-11ed-8ce5-7b34330a9d7c",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialrouteID: uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
						ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
					},
				},
			},

			responseProvider: &rmprovider.Provider{
				ID:       uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
				Hostname: "test.com",
			},

			expectProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
			expectArgs:       "context_type=call,context=call-out,call_id=7e0a846a-5d96-11ed-9005-07794a4f93cb,transport=udp,direction=outgoing",
			expectDialURI:    "pjsip/call-out/sip:+821100000001@test.com;transport=udp",
			expectVariables: map[string]string{
				"CALLERID(name)": "",
				"CALLERID(num)":  "+821100000002",
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)
			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, tt.call.ChannelID, tt.expectArgs, tt.expectDialURI, "", "", "", tt.expectVariables).Return(&channel.Channel{}, nil)

			if err := h.createChannelOutgoing(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_createFailoverChannel(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseUUIDChannel uuid.UUID
		responseCall        *call.Call
		responseProvider    *rmprovider.Provider

		expectDialrouteID uuid.UUID
		expectProviderID  uuid.UUID
		expectArgs        string
		expectDialURI     string
		expectVariables   map[string]string

		expectRes *call.Call
	}{
		{
			"normal",

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("25c7a29a-5f7d-11ed-86cc-bb999f3cccaf"),
					CustomerID: uuid.FromStringOrNil("260b56c0-5f7d-11ed-8930-cbd2b7cb46ff"),
				},

				ChannelID: "263bc88c-5f7d-11ed-a26c-735426e79b33",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialrouteID: uuid.FromStringOrNil("266caba0-5f7d-11ed-b8bc-c3e291d85720"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("266caba0-5f7d-11ed-b8bc-c3e291d85720"),
						ProviderID: uuid.FromStringOrNil("269dad7c-5f7d-11ed-aa09-c30aa616e3ea"),
					},
					{
						ID:         uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
						ProviderID: uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
					},
				},
			},

			uuid.FromStringOrNil("2902a512-5f7e-11ed-8cb0-ef43ac5ddea8"),
			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("25c7a29a-5f7d-11ed-86cc-bb999f3cccaf"),
					CustomerID: uuid.FromStringOrNil("260b56c0-5f7d-11ed-8930-cbd2b7cb46ff"),
				},

				ChannelID: "2902a512-5f7e-11ed-8cb0-ef43ac5ddea8",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialrouteID: uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("266caba0-5f7d-11ed-b8bc-c3e291d85720"),
						ProviderID: uuid.FromStringOrNil("269dad7c-5f7d-11ed-aa09-c30aa616e3ea"),
					},
					{
						ID:         uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
						ProviderID: uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
					},
				},
			},
			&rmprovider.Provider{
				ID:       uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
				Hostname: "test.com",
			},

			uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
			uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
			"context_type=call,context=call-out,call_id=25c7a29a-5f7d-11ed-86cc-bb999f3cccaf,transport=udp,direction=outgoing",
			"pjsip/call-out/sip:+821100000001@test.com;transport=udp",
			map[string]string{
				"CALLERID(name)": "",
				"CALLERID(num)":  "+821100000002",
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("25c7a29a-5f7d-11ed-86cc-bb999f3cccaf"),
					CustomerID: uuid.FromStringOrNil("260b56c0-5f7d-11ed-8930-cbd2b7cb46ff"),
				},

				ChannelID: "2902a512-5f7e-11ed-8cb0-ef43ac5ddea8",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialrouteID: uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("266caba0-5f7d-11ed-b8bc-c3e291d85720"),
						ProviderID: uuid.FromStringOrNil("269dad7c-5f7d-11ed-aa09-c30aa616e3ea"),
					},
					{
						ID:         uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
						ProviderID: uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
					},
				},
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

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)

			// updateForRouteFailover
			mockDB.EXPECT().CallSetForRouteFailover(ctx, tt.call.ID, tt.responseUUIDChannel.String(), tt.expectDialrouteID).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallUpdated, tt.responseCall)

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)
			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, tt.responseCall.ChannelID, tt.expectArgs, tt.expectDialURI, "", "", "", tt.expectVariables).Return(&channel.Channel{}, nil)

			res, err := h.createFailoverChannel(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getNextDialroute(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		expectRes *rmroute.Route
	}{
		{
			"normal",

			&call.Call{
				DialrouteID: uuid.FromStringOrNil("d1112a98-5f7f-11ed-944b-3f78d26bca39"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("d1112a98-5f7f-11ed-944b-3f78d26bca39"),
						ProviderID: uuid.FromStringOrNil("d14afce6-5f7f-11ed-baff-6b5dd84ca88b"),
					},
					{
						ID:         uuid.FromStringOrNil("d1798e58-5f7f-11ed-bb82-770f515db7bb"),
						ProviderID: uuid.FromStringOrNil("d1adc1be-5f7f-11ed-860e-efd0fa6f95c5"),
					},
				},
			},

			&rmroute.Route{
				ID:         uuid.FromStringOrNil("d1798e58-5f7f-11ed-bb82-770f515db7bb"),
				ProviderID: uuid.FromStringOrNil("d1adc1be-5f7f-11ed-860e-efd0fa6f95c5"),
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

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			res, err := h.getNextDialroute(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getNextDialroute_error(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call
	}{
		{
			"empty routes",

			&call.Call{
				DialrouteID: uuid.Nil,
				Dialroutes:  []rmroute.Route{},
			},
		},
		{
			"no more dialroutes left",

			&call.Call{
				DialrouteID: uuid.FromStringOrNil("8bd1aba0-5f80-11ed-84f3-13ad177e24a7"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("9c86e082-5f80-11ed-8345-2b30b7c6d70e"),
						ProviderID: uuid.FromStringOrNil("9cad5a1e-5f80-11ed-87a1-73640a77ce52"),
					},
					{
						ID:         uuid.FromStringOrNil("8bd1aba0-5f80-11ed-84f3-13ad177e24a7"),
						ProviderID: uuid.FromStringOrNil("9cf9531a-5f80-11ed-b3d4-5bc4b0bec0c2"),
					},
				},
			},
		},
		{
			"dialroute id does not exist in the dialroutes",

			&call.Call{
				DialrouteID: uuid.FromStringOrNil("b86c9170-5f80-11ed-9429-ff3d42d871e9"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("b8941d9e-5f80-11ed-ba54-f3c6180e2331"),
						ProviderID: uuid.FromStringOrNil("b8bc7244-5f80-11ed-b9c9-cb656548f971"),
					},
					{
						ID:         uuid.FromStringOrNil("b8e66464-5f80-11ed-be77-dffb61c583e3"),
						ProviderID: uuid.FromStringOrNil("b911a7be-5f80-11ed-a251-ef2d84919cb7"),
					},
				},
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

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			_, err := h.getNextDialroute(ctx, tt.call)
			if err == nil {
				t.Error("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_getDialroutes(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		destination *commonaddress.Address

		responseDialroutes []rmroute.Route

		expectTarget string

		expectRes []rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("551562fe-5f81-11ed-b9b3-535fe8d67b80"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},

			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("b8d6da7a-5f81-11ed-9274-9313db0184ad"),
				},
			},

			"+82",

			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("b8d6da7a-5f81-11ed-9274-9313db0184ad"),
				},
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

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1DialrouteList(ctx, gomock.Any()).Return(tt.responseDialroutes, nil)

			res, err := h.getDialroutes(ctx, tt.customerID, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_setChannelVariablesCallerID(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		expectRes map[string]string
	}{
		{
			"destination type tel and source target is anonymous",

			&call.Call{
				Source: commonaddress.Address{
					Target: "anonymous",
				},
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
			},

			map[string]string{
				"CALLERID(pres)":                        "prohib",
				"PJSIP_HEADER(add,P-Asserted-Identity)": "\"Anonymous\" <sip:+821100000001@pstn.voipbin.net>",
				"PJSIP_HEADER(add,Privacy)":             "id",
			},
		},
		{
			"destination type is sip",

			&call.Call{
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "sip:test@test.trunk.voipbin.net",
				},
			},

			map[string]string{
				"CALLERID(name)": "",
				"CALLERID(num)":  "+821100000001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := map[string]string{}
			setChannelVariablesCallerID(res, tt.call)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getSourceForOutgoingCall(t *testing.T) {

	tests := []struct {
		name string

		source      *commonaddress.Address
		destination *commonaddress.Address

		expectRes *commonaddress.Address
	}{
		{
			"destination type tel and source target is anonymous",

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
		},
		{
			"destination type is tel but source has + prefix",

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},

			&commonaddress.Address{
				Type:       commonaddress.TypeTel,
				TargetName: "Anonymous",
				Target:     "anonymous",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := getSourceForOutgoingCall(tt.source, tt.destination)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getGroupcallRingMethod_destination_type_agent(t *testing.T) {

	tests := []struct {
		name string

		destination commonaddress.Address

		responseAgent *amagent.Agent

		expectRes groupcall.RingMethod
	}{
		{
			name: "agent ring method ring linear",

			destination: commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "de4249b4-e278-11ed-adcd-0b3ffc5eafb6",
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("de4249b4-e278-11ed-adcd-0b3ffc5eafb6"),
				},
				RingMethod: amagent.RingMethodLinear,
			},

			expectRes: groupcall.RingMethodLinear,
		},
		{
			name: "agent ring method ring all",

			destination: commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "de806866-e278-11ed-83fd-17826e514dba",
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("de806866-e278-11ed-83fd-17826e514dba"),
				},
				RingMethod: amagent.RingMethodRingAll,
			},

			expectRes: groupcall.RingMethodRingAll,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				notifyHandler:    mockNotify,
				channelHandler:   mockChannel,
				groupcallHandler: mockGroupcall,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, uuid.FromStringOrNil(tt.destination.Target)).Return(tt.responseAgent, nil)

			res, err := h.getGroupcallRingMethod(ctx, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getDestinationTransport(t *testing.T) {

	tests := []struct {
		name string

		endpoint string

		expectRes channel.SIPTransport
	}{
		{
			name: "websocket",

			endpoint:  ";transport=ws",
			expectRes: channel.SIPTransportWS,
		},
		{
			name: "secured websocket",

			endpoint:  ";transport=wss",
			expectRes: channel.SIPTransportWSS,
		},
		{
			name: "tcp",

			endpoint:  ";transport=tcp",
			expectRes: channel.SIPTransportTCP,
		},
		{
			name: "tls",

			endpoint:  ";transport=tls",
			expectRes: channel.SIPTransportTLS,
		},
		{
			name: "udp",

			endpoint:  ";transport=udp",
			expectRes: channel.SIPTransportUDP,
		},
		{
			name: "default",

			endpoint:  ";transport=",
			expectRes: channel.SIPTransportUDP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := getDestinationTransport(tt.endpoint)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_setChannelVariableTransport(t *testing.T) {

	tests := []struct {
		name string

		variables map[string]string
		transport channel.SIPTransport

		expectRes map[string]string
	}{
		{
			name: "ws",

			variables: map[string]string{},
			transport: channel.SIPTransportWS,

			expectRes: map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "UDP/TLS/RTP/SAVPF",
			},
		},
		{
			name: "wss",

			variables: map[string]string{},
			transport: channel.SIPTransportWSS,

			expectRes: map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "UDP/TLS/RTP/SAVPF",
			},
		},
		{
			name: "tcp",

			variables: map[string]string{},
			transport: channel.SIPTransportTCP,

			expectRes: map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
			},
		},
		{
			name: "tls",

			variables: map[string]string{},
			transport: channel.SIPTransportTLS,

			expectRes: map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
			},
		},
		{
			name: "udp",

			variables: map[string]string{},
			transport: channel.SIPTransportUDP,

			expectRes: map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
			},
		},
		{
			name: "default",

			variables: map[string]string{},
			transport: channel.SIPTransportNone,

			expectRes: map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			setChannelVariableTransport(tt.variables, tt.transport)
			if !reflect.DeepEqual(tt.variables, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, tt.variables)
			}
		})
	}
}

func Test_getDialURISIP(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		expectRes string
	}{
		{
			name: "normal",

			call: &call.Call{
				Destination: commonaddress.Address{
					Target: "sip:3000@211.200.20.28:49699^3Btransport=udp^3Balias=211.200.20.28~49699~1",
				},
			},

			expectRes: "pjsip/call-out/sip:3000@211.200.20.28:49699^3Btransport=udp^3Balias=211.200.20.28~49699~1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				notifyHandler:    mockNotify,
				channelHandler:   mockChannel,
				groupcallHandler: mockGroupcall,
			}
			ctx := context.Background()

			res, err := h.getDialURISIP(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_getDialURISIPDirect(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		expectRes string
	}{
		{
			name: "normal",

			call: &call.Call{
				Destination: commonaddress.Address{
					Target: "sip:ind5v09k@3kssqpaa87pe.invalid;transport=ws;alias=35.204.215.63~36432~5;outbound_proxy=10.164.0.9",
				},
			},

			expectRes: "pjsip/call-out-direct-10.164.0.9/sip:ind5v09k@3kssqpaa87pe.invalid;transport=ws;alias=35.204.215.63~36432~5;outbound_proxy=10.164.0.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				notifyHandler:    mockNotify,
				channelHandler:   mockChannel,
				groupcallHandler: mockGroupcall,
			}
			ctx := context.Background()

			res, err := h.getDialURISIPDirect(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
