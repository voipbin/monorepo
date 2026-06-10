package callhandler

import (
	"monorepo/bin-call-manager/pkg/testhelper"
	"context"
	stderrors "errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	nmnumber "monorepo/bin-number-manager/models/number"

	rmprovider "monorepo/bin-route-manager/models/provider"
	rmroute "monorepo/bin-route-manager/models/route"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/models/groupcall"
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"
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

				ChainedCallIDs:  []uuid.UUID{},
				RecordingIDs:    []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},

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
					call.DataTypeAnonymous:                 "false",
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},

				Dialroutes: []rmroute.Route{},

				TMCreate: testhelper.TimePtr("2021-02-19T06:32:14.621Z"),
				TMUpdate:      nil,
				TMRinging:     nil,
				TMProgressing: nil,
				TMHangup:      nil,
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
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				utilHandler:           mockUtil,
				reqHandler:            mockReq,
				notifyHandler:         mockNotify,
				db:                    mockDB,
				channelHandler:        mockChannel,
				outboundConfigHandler: mockOutboundConfig,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.customerID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseActiveflow, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(&cucustomer.Customer{
				ID:                         tt.customerID,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(nil, nil)
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

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, tt.earlyExecution, tt.connect, "", nil, nil)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

// Test_CreateCallOutgoing_Metadata verifies that caller-supplied metadata is persisted
// verbatim on the Call record at creation time. This is the plumbing that Task 11 relies
// on so getDialroutes can forward targetProviderIDs to route-manager.
func Test_CreateCallOutgoing_Metadata(t *testing.T) {

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
		metadata       map[string]interface{}

		responseActiveflow  *fmactiveflow.Activeflow
		responseAgent       *amagent.Agent
		responseUUIDChannel uuid.UUID

		expectCall *call.Call
	}{
		{
			name: "metadata with route_provider_ids is persisted on Call",

			id:           uuid.FromStringOrNil("9e9c3b3e-0e8a-4a9d-9f5e-2ce6a0c63d77"),
			customerID:   uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
			flowID:       uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
			activeflowID: uuid.FromStringOrNil("679f0eb2-8c21-41a6-876d-9d778b1b0167"),
			masterCallID: uuid.Nil,
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
			metadata: map[string]interface{}{
				call.MetadataKeyRouteProviderIDs: []interface{}{
					"6a5c1b8e-11ef-4e3a-9a1b-abc123456789",
					"7f2d9c01-11ef-4b2a-8c3d-def987654321",
				},
			},

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

			expectCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9e9c3b3e-0e8a-4a9d-9f5e-2ce6a0c63d77"),
					CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("1aa075dc-2bfe-11ef-9203-37278cb94d16"),
				},
				ChannelID: "80d67b3a-5f3b-11ed-a709-0f2943ef0184",
				FlowID:    uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
				Type:      call.TypeFlow,

				ChainedCallIDs:   []uuid.UUID{},
				RecordingIDs:     []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},

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
					call.DataTypeAnonymous:                 "false",
				},
				Metadata: map[string]interface{}{
					call.MetadataKeyRouteProviderIDs: []interface{}{
						"6a5c1b8e-11ef-4e3a-9a1b-abc123456789",
						"7f2d9c01-11ef-4b2a-8c3d-def987654321",
					},
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},

				Dialroutes: []rmroute.Route{},
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
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				utilHandler:           mockUtil,
				reqHandler:            mockReq,
				notifyHandler:         mockNotify,
				db:                    mockDB,
				channelHandler:        mockChannel,
				outboundConfigHandler: mockOutboundConfig,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.customerID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseActiveflow, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(&cucustomer.Customer{
				ID:                         tt.customerID,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(nil, nil)
			mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.customerID, tt.destination).Return(tt.responseAgent, nil)

			// CRITICAL assertion: verify the Call passed to CallCreate carries the caller-supplied Metadata.
			// CallCreate expectation matches against tt.expectCall (Matches uses reflect.DeepEqual on all fields
			// including Metadata — see call.Call.Matches in models/call/call.go).
			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)
			mockReq.EXPECT().CallV1CallHealth(ctx, tt.expectCall.ID, defaultHealthDelay, 0).Return(nil)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, gomock.Any(), gomock.Any(), gomock.Any(), "", "", "", gomock.Any()).Return(&channel.Channel{}, nil)

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, tt.earlyExecution, tt.connect, "", tt.metadata, nil)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil {
				t.Fatalf("Wrong match. expected non-nil call result")
			}
			if !reflect.DeepEqual(res.Metadata, tt.expectCall.Metadata) {
				t.Errorf("Wrong match. Metadata differs.\nexpect: %v\ngot:    %v", tt.expectCall.Metadata, res.Metadata)
			}
		})
	}
}

// Test_CreateCallOutgoing_RTPDebug verifies that when a customer has RTPDebug enabled,
// the call's Metadata map is populated with rtp_debug=true at creation time.
// It also verifies that a pre-set rtp_debug=true (e.g., from providercallhandler) is
// preserved even when the customer has RTPDebug=false.
func Test_CreateCallOutgoing_RTPDebug(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		activeflowID uuid.UUID
		masterCallID uuid.UUID
		source       commonaddress.Address
		destination  commonaddress.Address
		metadata     map[string]interface{} // incoming metadata passed to CreateCallOutgoing

		responseActiveflow  *fmactiveflow.Activeflow
		responseAgent       *amagent.Agent
		responseUUIDChannel uuid.UUID
		responseCustomer    *cucustomer.Customer

		expectCall *call.Call
	}{
		{
			name: "customer rtp_debug true embeds metadata key at creation",

			id:           uuid.FromStringOrNil("a1b2c3d4-ecb2-11ea-ab94-a768ab787da0"),
			customerID:   uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
			flowID:       uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
			activeflowID: uuid.FromStringOrNil("679f0eb2-8c21-41a6-876d-9d778b1b0167"),
			masterCallID: uuid.Nil,
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
			responseCustomer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
				Metadata: cucustomer.Metadata{
					RTPDebug: true,
				},
			},

			expectCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-ecb2-11ea-ab94-a768ab787da0"),
					CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("1aa075dc-2bfe-11ef-9203-37278cb94d16"),
				},
				ChannelID: "80d67b3a-5f3b-11ed-a709-0f2943ef0184",
				FlowID:    uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
				Type:      call.TypeFlow,

				ChainedCallIDs:   []uuid.UUID{},
				RecordingIDs:     []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},

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
					call.DataTypeEarlyExecution:            "false",
					call.DataTypeExecuteNextMasterOnHangup: "true",
					call.DataTypeAnonymous:                 "false",
				},
				// rtp_debug must be embedded in Metadata at creation time
				Metadata: map[string]any{
					call.MetadataKeyRTPDebug: true,
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},

				Dialroutes: []rmroute.Route{},
			},
		},
		{
			name: "provider call pre-sets rtp_debug=true, customer has RTPDebug=false — flag is preserved",

			id:           uuid.FromStringOrNil("b2c3d4e5-ecb2-11ea-ab94-a768ab787da0"),
			customerID:   uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
			flowID:       uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
			activeflowID: uuid.FromStringOrNil("679f0eb2-8c21-41a6-876d-9d778b1b0167"),
			masterCallID: uuid.Nil,
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
			// Simulate what providercallhandler sets: rtp_debug forced to true
			metadata: map[string]interface{}{
				call.MetadataKeyRTPDebug: true,
			},

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
			responseUUIDChannel: uuid.FromStringOrNil("90e78c4b-5f3b-11ed-a709-0f2943ef0184"),
			// Customer has RTPDebug=false — the provider-set flag must NOT be cleared
			responseCustomer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
				Metadata: cucustomer.Metadata{
					RTPDebug: false,
				},
			},

			expectCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2c3d4e5-ecb2-11ea-ab94-a768ab787da0"),
					CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("1aa075dc-2bfe-11ef-9203-37278cb94d16"),
				},
				ChannelID: "90e78c4b-5f3b-11ed-a709-0f2943ef0184",
				FlowID:    uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
				Type:      call.TypeFlow,

				ChainedCallIDs:   []uuid.UUID{},
				RecordingIDs:     []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},

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
					call.DataTypeEarlyExecution:            "false",
					call.DataTypeExecuteNextMasterOnHangup: "true",
					call.DataTypeAnonymous:                 "false",
				},
				// rtp_debug must be preserved even though customer has RTPDebug=false
				Metadata: map[string]any{
					call.MetadataKeyRTPDebug: true,
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},

				Dialroutes: []rmroute.Route{},
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
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				utilHandler:           mockUtil,
				reqHandler:            mockReq,
				notifyHandler:         mockNotify,
				db:                    mockDB,
				channelHandler:        mockChannel,
				outboundConfigHandler: mockOutboundConfig,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.customerID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseActiveflow, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(nil, nil)
			mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.customerID, tt.destination).Return(tt.responseAgent, nil)

			// CRITICAL assertion: CallCreate must receive the call with rtp_debug in Metadata.
			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)
			mockReq.EXPECT().CallV1CallHealth(ctx, tt.expectCall.ID, defaultHealthDelay, 0).Return(nil)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, gomock.Any(), gomock.Any(), gomock.Any(), "", "", "", gomock.Any()).Return(&channel.Channel{}, nil)

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, false, true, "", tt.metadata, nil)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil {
				t.Fatalf("Wrong match. expected non-nil call result")
			}
			if !reflect.DeepEqual(res.Metadata, tt.expectCall.Metadata) {
				t.Errorf("Wrong match. Metadata differs.\nexpect: %v\ngot:    %v", tt.expectCall.Metadata, res.Metadata)
			}
			if res.Metadata[call.MetadataKeyRTPDebug] != true {
				t.Errorf("Wrong match. expected rtp_debug=true in metadata, got: %v", res.Metadata[call.MetadataKeyRTPDebug])
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
				ChainedCallIDs:  []uuid.UUID{},
				RecordingIDs:    []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},
				Status:          call.StatusDialing,
				Data: map[call.DataType]string{
					call.DataTypeEarlyExecution:            "true",
					call.DataTypeExecuteNextMasterOnHangup: "true",
					call.DataTypeAnonymous:                 "false",
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

				TMCreate: testhelper.TimePtr("2021-02-19T06:32:14.621Z"),
				TMUpdate:      nil,
				TMRinging:     nil,
				TMProgressing: nil,
				TMHangup:      nil,
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
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				utilHandler:           mockUtil,
				reqHandler:            mockReq,
				notifyHandler:         mockNotify,
				db:                    mockDB,
				channelHandler:        mockChannel,
				outboundConfigHandler: mockOutboundConfig,
			}

			ctx := context.Background()

			// outbound config: return a permissive config (KR in whitelist) for tel destination
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(&outboundconfig.OutboundConfig{
				DestinationWhitelist: []string{"kr"},
			}, nil)

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.customerID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseActiveflow, nil)
			// getDialURI
			mockReq.EXPECT().RouteV1DialrouteList(ctx, gomock.Any(), gomock.Any()).Return(tt.responseRoutes, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(&cucustomer.Customer{
				ID:                         tt.customerID,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)

			// source number validation: source +99999888 belongs to customer as a normal number
			mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
				nmnumber.FieldCustomerID: tt.customerID,
				nmnumber.FieldNumber:     tt.source.Target,
				nmnumber.FieldType:       nmnumber.TypeNormal,
				nmnumber.FieldStatus:     nmnumber.StatusActive,
				nmnumber.FieldDeleted:    false,
			}).Return([]nmnumber.Number{{Number: tt.source.Target}}, nil)

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

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, tt.earlyExecution, tt.connect, "", nil, nil)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

// Test_CreateCallOutgoing_TypeTel_OutboundConfigFetchError_FailClosed locks in the
// fail-closed contract: when outboundConfigHandler.GetByCustomerID returns a transient
// error for a tel destination on a non-internal customer, CreateCallOutgoing MUST
// reject the call (return nil, err) instead of silently continuing with outboundCfg=nil.
//
// Regression guard for the previous "log warn + outboundCfg = nil + continue" behavior
// that allowed calls to bypass whitelist enforcement when the outbound_config DB read
// failed transiently. See outgoing_call.go fail-closed branch.
func Test_CreateCallOutgoing_TypeTel_OutboundConfigFetchError_FailClosed(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		activeflowID uuid.UUID
		masterCallID uuid.UUID
		source       commonaddress.Address
		destination  commonaddress.Address

		responseCustomer *cucustomer.Customer
		fetchErr         error
	}{
		{
			name: "outbound config fetch returns transient db error - call rejected",

			id:           uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000001"),
			customerID:   uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-0000000000c1"),
			flowID:       uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-0000000000f1"),
			activeflowID: uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-0000000000a1"),
			masterCallID: uuid.Nil,
			source: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+14155550100",
				TargetName: "test",
			},
			destination: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+14155550199",
				TargetName: "test target",
			},

			responseCustomer: &cucustomer.Customer{
				ID:                         uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-0000000000c1"),
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			},
			fetchErr: fmt.Errorf("transient db error"),
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
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				utilHandler:           mockUtil,
				reqHandler:            mockReq,
				notifyHandler:         mockNotify,
				db:                    mockDB,
				channelHandler:        mockChannel,
				outboundConfigHandler: mockOutboundConfig,
			}

			ctx := context.Background()

			// customer fetch (precedes the outbound config fetch) succeeds
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
			// balance check passes so the function reaches the OutboundConfig fetch site
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)

			// the load-bearing mock: outbound config fetch fails
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(nil, tt.fetchErr)

			// no further interactions: dialroutes, activeflow create, channel start, etc.
			// must NOT be invoked. gomock will fail the test if any unexpected call lands.

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, false, false, "", nil, nil)

			// fail-closed: must return (nil, error)
			if res != nil {
				t.Errorf("Wrong match. expect: nil call, got: %v", res)
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}

			// error must wrap the underlying transient db error (errors.Is should walk the
			// %w chain — guards against accidentally losing provenance via fmt.Errorf("%v")).
			if !stderrors.Is(err, tt.fetchErr) {
				t.Errorf("Wrong match. expect error to wrap %v via %%w, got: %v", tt.fetchErr, err)
			}

			// error message must carry the "could not get outbound config" provenance so
			// operators can grep production logs for the fail-closed reason.
			if !strings.Contains(err.Error(), "could not get outbound config") {
				t.Errorf("Wrong match. expect error to mention 'could not get outbound config', got: %v", err)
			}
		})
	}
}

// Test_embedCodecs_ByDestinationType verifies that embedCodecs is called for SIP
// destinations and skipped for PSTN destinations.
func Test_embedCodecs_ByDestinationType(t *testing.T) {
	tests := []struct {
		name string

		destination commonaddress.Address
		outboundCfg *outboundconfig.OutboundConfig

		expectCodecInMetadata bool
	}{
		{
			name: "SIP destination with codecs set - codec embedded in metadata",

			destination: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testoutgoing@test.com",
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				Codecs:     "ulaw,alaw",
			},
			expectCodecInMetadata: true,
		},
		{
			name: "PSTN destination with codecs set - codec NOT embedded in metadata",

			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821121656521",
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				CustomerID: uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
				Codecs:     "ulaw,alaw",
			},
			expectCodecInMetadata: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := map[string]any{}
			switch tt.destination.Type {
			case commonaddress.TypeSIP:
				metadata = embedCodecs(metadata, tt.outboundCfg)
			case commonaddress.TypeTel:
				// no codec embedding for PSTN
			}

			_, hasCodec := metadata[call.MetadataKeyCodecs]
			if hasCodec != tt.expectCodecInMetadata {
				t.Errorf("Wrong match. codec in metadata = %v, want %v", hasCodec, tt.expectCodecInMetadata)
			}
		})
	}
}

// Test_CreateCallOutgoing_TypeSIP_OutboundConfigFetchError_FailClosed verifies
// that a DB error during outbound config fetch rejects a SIP call (fail-closed).
func Test_CreateCallOutgoing_TypeSIP_OutboundConfigFetchError_FailClosed(t *testing.T) {
	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		activeflowID uuid.UUID
		masterCallID uuid.UUID
		source       commonaddress.Address
		destination  commonaddress.Address
		fetchErr     error
	}{
		{
			name: "outbound config fetch returns db error for SIP call - call rejected",

			id:           uuid.FromStringOrNil("b1b2c3d4-0000-4000-8000-000000000001"),
			customerID:   uuid.FromStringOrNil("b1b2c3d4-0000-4000-8000-0000000000c1"),
			flowID:       uuid.FromStringOrNil("b1b2c3d4-0000-4000-8000-0000000000f1"),
			activeflowID: uuid.FromStringOrNil("b1b2c3d4-0000-4000-8000-0000000000a1"),
			masterCallID: uuid.Nil,
			source: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsrc@test.com",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testoutgoing@test.com",
			},
			fetchErr: fmt.Errorf("transient db error"),
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
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				utilHandler:           mockUtil,
				reqHandler:            mockReq,
				notifyHandler:         mockNotify,
				db:                    mockDB,
				channelHandler:        mockChannel,
				outboundConfigHandler: mockOutboundConfig,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(&cucustomer.Customer{
				ID:                         tt.customerID,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(nil, tt.fetchErr)

			// no further interactions: dialroutes, activeflow create, channel start, etc.
			// must NOT be invoked. gomock will fail the test if any unexpected call lands.

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, false, false, "", nil, nil)

			if res != nil {
				t.Errorf("Wrong match. expect: nil call, got: %v", res)
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !stderrors.Is(err, tt.fetchErr) {
				t.Errorf("Wrong match. expect error to wrap %v, got: %v", tt.fetchErr, err)
			}
			if !strings.Contains(err.Error(), "could not get outbound config") {
				t.Errorf("Wrong match. expect 'could not get outbound config' in error, got: %v", err)
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

			mockGroupcall.EXPECT().Start(ctx, uuid.Nil, tt.customerID, tt.flowID, tt.source, []commonaddress.Address{*tt.destination}, tt.masterCallID, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", gomock.Any()).Return(tt.responseGroupcall, nil)

			res, err := h.createCallsOutgoingGroupcall(ctx, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destination, "", nil)
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

			mockGroupcall.EXPECT().Start(ctx, uuid.Nil, tt.customerID, tt.flowID, tt.source, []commonaddress.Address{*tt.destination}, tt.masterCallID, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", gomock.Any()).Return(tt.responseGroupcall, nil)

			res, err := h.createCallsOutgoingGroupcall(ctx, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destination, "", nil)
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
		expectRes        string
		expectTechHdrs   map[string]string
		expectErr        bool
	}{
		{
			"no tech config (backwards compat)",

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
			"pjsip/call-out/sip:+821121656521@sip.telnyx.com;transport=udp",
			nil,
			false,
		},
		{
			"prefix only",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
						ProviderID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001")},
				},
			},

			&rmprovider.Provider{
				Hostname:   "carrier.example.com",
				TechPrefix: "0011",
			},

			uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001"),
			"pjsip/call-out/sip:001115551234@carrier.example.com;transport=udp",
			nil,
			false,
		},
		{
			"postfix only",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000002"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000002"),
						ProviderID: uuid.FromStringOrNil("b0000002-0000-0000-0000-000000000002")},
				},
			},

			&rmprovider.Provider{
				Hostname:    "carrier.example.com",
				TechPostfix: "#",
			},

			uuid.FromStringOrNil("b0000002-0000-0000-0000-000000000002"),
			"pjsip/call-out/sip:+15551234#@carrier.example.com;transport=udp",
			nil,
			false,
		},
		{
			"prefix and postfix both",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000003-0000-0000-0000-000000000003"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000003-0000-0000-0000-000000000003"),
						ProviderID: uuid.FromStringOrNil("b0000003-0000-0000-0000-000000000003")},
				},
			},

			&rmprovider.Provider{
				Hostname:    "carrier.example.com",
				TechPrefix:  "0011",
				TechPostfix: "#",
			},

			uuid.FromStringOrNil("b0000003-0000-0000-0000-000000000003"),
			"pjsip/call-out/sip:0011+15551234#@carrier.example.com;transport=udp",
			nil,
			false,
		},
		{
			"headers only — returned raw (unsanitized), caller sanitizes via mergeTechHeaders",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000004-0000-0000-0000-000000000004"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000004-0000-0000-0000-000000000004"),
						ProviderID: uuid.FromStringOrNil("b0000004-0000-0000-0000-000000000004")},
				},
			},

			&rmprovider.Provider{
				Hostname: "carrier.example.com",
				TechHeaders: map[string]string{
					"X-Carrier-Auth": "tok-abc",
				},
			},

			uuid.FromStringOrNil("b0000004-0000-0000-0000-000000000004"),
			"pjsip/call-out/sip:+15551234@carrier.example.com;transport=udp",
			map[string]string{"X-Carrier-Auth": "tok-abc"},
			false,
		},
		{
			"all three together",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000005-0000-0000-0000-000000000005"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000005-0000-0000-0000-000000000005"),
						ProviderID: uuid.FromStringOrNil("b0000005-0000-0000-0000-000000000005")},
				},
			},

			&rmprovider.Provider{
				Hostname:    "carrier.example.com",
				TechPrefix:  "0011",
				TechPostfix: "#",
				TechHeaders: map[string]string{"X-Route-Hint": "premium"},
			},

			uuid.FromStringOrNil("b0000005-0000-0000-0000-000000000005"),
			"pjsip/call-out/sip:001115551234#@carrier.example.com;transport=udp",
			map[string]string{"X-Route-Hint": "premium"},
			false,
		},
		{
			"provider fetch error — returns err with nil tech_headers",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000006"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000006"),
						ProviderID: uuid.FromStringOrNil("b0000006-0000-0000-0000-000000000006")},
				},
			},

			nil,

			uuid.FromStringOrNil("b0000006-0000-0000-0000-000000000006"),
			"",
			nil,
			true,
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

			if tt.expectErr {
				mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(nil, fmt.Errorf("mock provider-fetch failure"))
			} else {
				mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)
			}

			target, err := h.getDialURI(ctx, tt.call)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: err, got: ok")
				}
				if target != nil {
					t.Errorf("Wrong target on error. expect: nil, got: %v", target)
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if target.URI != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, target.URI)
			}

			if len(target.TechHeaders) != len(tt.expectTechHdrs) {
				t.Errorf("Wrong techHdrs size. expect: %d, got: %d. techHdrs=%v", len(tt.expectTechHdrs), len(target.TechHeaders), target.TechHeaders)
			}
			for k, v := range tt.expectTechHdrs {
				if got, ok := target.TechHeaders[k]; !ok || got != v {
					t.Errorf("Wrong techHdrs entry. key=%s expect=%q got=%q (present=%v)", k, v, got, ok)
				}
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

			target, err := h.getDialURI(ctx, tt.c)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if target.URI != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, target.URI)
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
		{
			name: "provider tech config applied — prefix, postfix, header",

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000002"),
				},

				ChannelID: "c1c1c1c1-0000-0000-0000-000000000003",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "15551234",
				},
				DialrouteID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000004"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000004"),
						ProviderID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000005"),
					},
				},
			},

			responseProvider: &rmprovider.Provider{
				ID:          uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000005"),
				Hostname:    "carrier.example.com",
				TechPrefix:  "0011",
				TechPostfix: "#",
				TechHeaders: map[string]string{"X-Route-Hint": "premium"},
			},

			expectProviderID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000005"),
			expectArgs:       "context_type=call,context=call-out,call_id=c1c1c1c1-0000-0000-0000-000000000001,transport=udp,direction=outgoing",
			expectDialURI:    "pjsip/call-out/sip:001115551234#@carrier.example.com;transport=udp",
			expectVariables: map[string]string{
				"PJSIP_HEADER(add,X-Route-Hint)":                         "premium",
				"CALLERID(name)":                                         "",
				"CALLERID(num)":                                          "+821100000002",
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
		metadata    map[string]interface{}

		responseDialroutes []rmroute.Route

		expectTarget            string
		expectTargetProviderIDs []uuid.UUID

		expectRes []rmroute.Route
	}{
		{
			name: "normal without metadata",

			customerID: uuid.FromStringOrNil("551562fe-5f81-11ed-b9b3-535fe8d67b80"),
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			metadata: nil,

			responseDialroutes: []rmroute.Route{
				{
					ID: uuid.FromStringOrNil("b8d6da7a-5f81-11ed-9274-9313db0184ad"),
				},
			},

			expectTarget:            "+82",
			expectTargetProviderIDs: nil,

			expectRes: []rmroute.Route{
				{
					ID: uuid.FromStringOrNil("b8d6da7a-5f81-11ed-9274-9313db0184ad"),
				},
			},
		},
		{
			name: "metadata without route_provider_ids forwards nil",

			customerID: uuid.FromStringOrNil("551562fe-5f81-11ed-b9b3-535fe8d67b80"),
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			metadata: map[string]interface{}{
				"some_other_key": "some_other_value",
			},

			responseDialroutes: []rmroute.Route{
				{
					ID: uuid.FromStringOrNil("b8d6da7a-5f81-11ed-9274-9313db0184ad"),
				},
			},

			expectTarget:            "+82",
			expectTargetProviderIDs: nil,

			expectRes: []rmroute.Route{
				{
					ID: uuid.FromStringOrNil("b8d6da7a-5f81-11ed-9274-9313db0184ad"),
				},
			},
		},
		{
			name: "route_provider_ids metadata is parsed and forwarded as targetProviderIDs",

			customerID: uuid.FromStringOrNil("551562fe-5f81-11ed-b9b3-535fe8d67b80"),
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			metadata: map[string]interface{}{
				call.MetadataKeyRouteProviderIDs: []interface{}{
					"11111111-1111-1111-1111-111111111111",
					"22222222-2222-2222-2222-222222222222",
				},
			},

			responseDialroutes: []rmroute.Route{
				{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					ProviderID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
				{
					ID:         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					ProviderID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
			},

			expectTarget: "+82",
			expectTargetProviderIDs: []uuid.UUID{
				uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			},

			expectRes: []rmroute.Route{
				{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					ProviderID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
				{
					ID:         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					ProviderID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
			},
		},
		{
			name: "invalid UUID in route_provider_ids is skipped",

			customerID: uuid.FromStringOrNil("551562fe-5f81-11ed-b9b3-535fe8d67b80"),
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			metadata: map[string]interface{}{
				call.MetadataKeyRouteProviderIDs: []interface{}{
					"not-a-uuid",
					"33333333-3333-3333-3333-333333333333",
				},
			},

			responseDialroutes: []rmroute.Route{
				{
					ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
					ProviderID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				},
			},

			expectTarget: "+82",
			expectTargetProviderIDs: []uuid.UUID{
				uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
			},

			expectRes: []rmroute.Route{
				{
					ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
					ProviderID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
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

			expectedFilters := map[rmroute.Field]any{
				rmroute.FieldCustomerID: tt.customerID,
				rmroute.FieldTarget:     tt.expectTarget,
			}
			mockReq.EXPECT().RouteV1DialrouteList(ctx, expectedFilters, tt.expectTargetProviderIDs).Return(tt.responseDialroutes, nil)

			res, err := h.getDialroutes(ctx, tt.customerID, tt.destination, tt.metadata)
			if err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_getDialroutes_errorCases verifies that malformed route_provider_ids metadata
// fails fast instead of silently falling through to normal routing.
func Test_getDialroutes_errorCases(t *testing.T) {

	tests := []struct {
		name        string
		customerID  uuid.UUID
		destination *commonaddress.Address
		metadata    map[string]interface{}
	}{
		{
			name:       "route_provider_ids with only invalid UUIDs returns error (no silent fallback)",
			customerID: uuid.FromStringOrNil("551562fe-5f81-11ed-b9b3-535fe8d67b80"),
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			metadata: map[string]interface{}{
				call.MetadataKeyRouteProviderIDs: []interface{}{
					"not-a-uuid",
					"also-not-a-uuid",
				},
			},
		},
		{
			name:       "route_provider_ids with wrong top-level type returns error",
			customerID: uuid.FromStringOrNil("551562fe-5f81-11ed-b9b3-535fe8d67b80"),
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			metadata: map[string]interface{}{
				call.MetadataKeyRouteProviderIDs: "not-an-array",
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

			// mockReq.EXPECT is NOT set — RouteV1DialrouteList must NOT be called.
			// gomock will fail the test if it is.

			res, err := h.getDialroutes(ctx, tt.customerID, tt.destination, tt.metadata)
			if err == nil {
				t.Errorf("Expected error for malformed route_provider_ids, got nil")
			}
			if res != nil {
				t.Errorf("Expected nil result, got: %v", res)
			}
		})
	}
}

func Test_setChannelVariablesCallerID(t *testing.T) {

	tests := []struct {
		name string

		call      *call.Call
		anonymous bool

		expectRes map[string]string
		expectErr bool
	}{
		{
			"destination type tel and anonymous true",

			&call.Call{
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "Test User",
				},
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
			},
			true,

			map[string]string{
				"CALLERID(name)":                        "Anonymous",
				"CALLERID(num)":                         "anonymous",
				"CALLERID(pres)":                        "prohib",
				"PJSIP_HEADER(add,P-Asserted-Identity)": "<tel:+821100000001>",
				"PJSIP_HEADER(add,Privacy)":             "id",
			},
			false,
		},
		{
			"destination type sip and anonymous true falls through to normal",

			&call.Call{
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "Test User",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "sip:test@test.trunk.voipbin.net",
				},
			},
			true,

			map[string]string{
				"CALLERID(name)": "Test User",
				"CALLERID(num)":  "+821100000001",
			},
			false,
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
			false,

			map[string]string{
				"CALLERID(name)": "",
				"CALLERID(num)":  "+821100000001",
			},
			false,
		},
		{
			"anonymous true with tel destination but empty source returns error",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
				},
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "",
					TargetName: "Empty Source",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821087654321",
				},
			},
			true,

			map[string]string{},
			true,
		},
		{
			"anonymous true with tel destination but non-E164 source returns error",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000002"),
				},
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "01012345678",
					TargetName: "Local Number",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821087654321",
				},
			},
			true,

			map[string]string{},
			true,
		},
		{
			"anonymous false with tel destination uses normal caller ID",

			&call.Call{
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "Normal Caller",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821087654321",
				},
			},
			false,

			map[string]string{
				"CALLERID(name)": "Normal Caller",
				"CALLERID(num)":  "+821100000001",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := map[string]string{}
			err := setChannelVariablesCallerID(res, tt.call, tt.anonymous)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getValidatedSourceForOutgoingCall(t *testing.T) {
	defaultNumberID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")

	tests := []struct {
		name string

		source      commonaddress.Address
		destination commonaddress.Address
		customer    *cucustomer.Customer
		outboundCfg *outboundconfig.OutboundConfig
		metadata    map[string]interface{}

		// expected NumberV1NumberList for the caller-supplied source path (uses Number= filter)
		responseNumbers []nmnumber.Number
		responseNumErr  error

		// expected NumberV1NumberList for the OutboundConfig fallback path (uses ID= filter)
		responseDefaultNumbers []nmnumber.Number
		responseDefaultNumErr  error

		expectRes *commonaddress.Address
	}{
		{
			name: "valid source owned by customer",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			responseNumbers: []nmnumber.Number{{Number: "+821100000001"}},
			expectRes: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
		},
		{
			name: "source not in E.164 format with cfg default available",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				DefaultOutgoingSourceNumberID: defaultNumberID,
			},
			responseDefaultNumbers: []nmnumber.Number{{Number: "+821100000099"}},
			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000099",
				TargetName: "+821100000099",
			},
		},
		{
			name: "source not owned by customer with cfg default available",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				DefaultOutgoingSourceNumberID: defaultNumberID,
			},
			responseNumbers:        []nmnumber.Number{},
			responseDefaultNumbers: []nmnumber.Number{{Number: "+821100000099"}},
			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000099",
				TargetName: "+821100000099",
			},
		},
		{
			name: "number lookup error with cfg default available",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				DefaultOutgoingSourceNumberID: defaultNumberID,
			},
			responseNumErr:         fmt.Errorf("number service error"),
			responseDefaultNumbers: []nmnumber.Number{{Number: "+821100000099"}},
			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000099",
				TargetName: "+821100000099",
			},
		},
		{
			name: "source not E.164 and nil cfg",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			outboundCfg: nil,
			expectRes:   nil,
		},
		{
			name: "source not owned by customer with cfg DefaultOutgoingSourceNumberID == uuid.Nil",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			outboundCfg:     &outboundconfig.OutboundConfig{}, // DefaultOutgoingSourceNumberID == uuid.Nil
			responseNumbers: []nmnumber.Number{},
			expectRes:       nil,
		},
		{
			name: "number lookup error and nil cfg",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			responseNumErr: fmt.Errorf("number service error"),
			outboundCfg:    nil,
			expectRes:      nil,
		},
		{
			name: "regression - default number released (NumberV1NumberList returns empty)",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				DefaultOutgoingSourceNumberID: defaultNumberID,
			},
			responseNumbers:        []nmnumber.Number{},
			responseDefaultNumbers: []nmnumber.Number{}, // released → re-validation returns empty
			expectRes:              nil,
		},
		{
			name: "default number re-validation errors out",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				DefaultOutgoingSourceNumberID: defaultNumberID,
			},
			responseNumbers:       []nmnumber.Number{},
			responseDefaultNumErr: fmt.Errorf("number list error"),
			expectRes:             nil,
		},
		{
			name: "nil customer skips validation",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: nil,
			expectRes: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
		},
		{
			name: "non-tel destination skips validation",
			source: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test@example.com",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "dest@example.com",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			expectRes: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test@example.com",
			},
		},
		{
			name: "skip_source_validation=true preserves unowned source (no mocks called)",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15559999999",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			outboundCfg: &outboundconfig.OutboundConfig{
				DefaultOutgoingSourceNumberID: defaultNumberID,
			},
			metadata: map[string]interface{}{
				call.MetadataKeySkipSourceValidation: true,
			},
			expectRes: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+15559999999",
			},
		},
		{
			name: "skip_source_validation=true preserves non-E.164 source",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "5551234",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			metadata: map[string]interface{}{
				call.MetadataKeySkipSourceValidation: true,
			},
			expectRes: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "5551234",
			},
		},
		{
			name: "skip_source_validation=false falls through to ownership validation",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			metadata: map[string]interface{}{
				call.MetadataKeySkipSourceValidation: false,
			},
			responseNumbers: []nmnumber.Number{{Number: "+821100000001"}},
			expectRes: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
		},
		{
			name: "skip_source_validation with non-bool value falls through to ownership validation",
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			},
			metadata: map[string]interface{}{
				call.MetadataKeySkipSourceValidation: "true",
			},
			responseNumbers: []nmnumber.Number{{Number: "+821100000001"}},
			expectRes: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
			}

			ctx := context.Background()

			// set up mocks based on test case.
			// when skip_source_validation is true, the function short-circuits before
			// any number-manager RPC is made — so we must NOT register expectations
			// that gomock would flag as unfulfilled.
			skipValidation := false
			if v, ok := tt.metadata[call.MetadataKeySkipSourceValidation].(bool); ok && v {
				skipValidation = true
			}

			// caller-supplied source path: only fires when source has E.164 prefix
			if !skipValidation && tt.customer != nil && tt.destination.Type == commonaddress.TypeTel && strings.HasPrefix(tt.source.Target, "+") {
				mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
					nmnumber.FieldCustomerID: tt.customer.ID,
					nmnumber.FieldNumber:     tt.source.Target,
					nmnumber.FieldType:       nmnumber.TypeNormal,
					nmnumber.FieldStatus:     nmnumber.StatusActive,
					nmnumber.FieldDeleted:    false,
				}).Return(tt.responseNumbers, tt.responseNumErr)
			}

			// fallback re-validation path: only fires when caller-supplied path failed
			// AND outboundCfg has a non-nil DefaultOutgoingSourceNumberID. The function
			// returns early when cfg is nil or its DefaultOutgoingSourceNumberID is uuid.Nil.
			callerPathSucceeded := strings.HasPrefix(tt.source.Target, "+") && len(tt.responseNumbers) > 0 && tt.responseNumErr == nil
			if !skipValidation && tt.customer != nil && tt.destination.Type == commonaddress.TypeTel &&
				!callerPathSucceeded &&
				tt.outboundCfg != nil && tt.outboundCfg.DefaultOutgoingSourceNumberID != uuid.Nil {
				mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
					nmnumber.FieldCustomerID: tt.customer.ID,
					nmnumber.FieldID:         tt.outboundCfg.DefaultOutgoingSourceNumberID,
					nmnumber.FieldType:       nmnumber.TypeNormal,
					nmnumber.FieldStatus:     nmnumber.StatusActive,
					nmnumber.FieldDeleted:    false,
				}).Return(tt.responseDefaultNumbers, tt.responseDefaultNumErr)
			}

			res := h.getValidatedSourceForOutgoingCall(ctx, tt.source, tt.destination, tt.customer, tt.outboundCfg, tt.metadata)
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

			target, err := h.getDialURISIP(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, target.URI) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, target.URI)
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

			target, err := h.getDialURISIPDirect(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, target.URI) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, target.URI)
			}

		})
	}
}

// Test_createChannelOutgoing_ProviderCodecs verifies that when a provider has a Codecs
// value, setProviderCodecs injects VBOUT-CODECS into the channel variables passed to
// StartChannel; and that when Codecs is empty, VBOUT-CODECS is absent.
func Test_createChannelOutgoing_ProviderCodecs(t *testing.T) {
	codecHeaderKey := "PJSIP_HEADER(add," + common.SIPHeaderCodecs + ")"

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

		expectProviderID   uuid.UUID
		expectCall         *call.Call
		expectEndpointDst  string
		expectArgs         string
		expectCodecPresent bool   // whether VBOUT-CODECS key should be in channel variables
		expectCodecValue   string // expected value when expectCodecPresent is true
	}{
		{
			name: "provider with codecs PCMU - VBOUT-CODECS injected into channel variables",

			id:           uuid.FromStringOrNil("c1c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
			customerID:   uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
			flowID:       uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
			activeflowID: uuid.FromStringOrNil("21e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
			masterCallID: uuid.Nil,
			source: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+14155550100",
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
					ID: uuid.FromStringOrNil("21e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
				},
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
			responseRoutes: []rmroute.Route{
				{
					ID:         uuid.FromStringOrNil("a86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
					ProviderID: uuid.FromStringOrNil("b213af44-534e-11ed-9a1d-73b0076723b8"),
				},
			},
			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1b095188-2bfe-11ef-a746-7f4de3b06e46"),
				},
			},
			responseUUIDChannel: uuid.FromStringOrNil("e948969e-5de3-11ed-94f5-137ec429b6b6"),
			responseProvider: &rmprovider.Provider{
				ID:       uuid.FromStringOrNil("b213af44-534e-11ed-9a1d-73b0076723b8"),
				Hostname: "sip.telnyx.com",
				Codecs:   "PCMU",
			},

			expectProviderID: uuid.FromStringOrNil("b213af44-534e-11ed-9a1d-73b0076723b8"),
			expectCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
					CustomerID: uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("1b095188-2bfe-11ef-a746-7f4de3b06e46"),
				},
				ChannelID:        "e948969e-5de3-11ed-94f5-137ec429b6b6",
				FlowID:           uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
				ActiveflowID:     uuid.FromStringOrNil("21e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
				Type:             call.TypeFlow,
				ChainedCallIDs:   []uuid.UUID{},
				RecordingIDs:     []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},
				Status:           call.StatusDialing,
				Data: map[call.DataType]string{
					call.DataTypeEarlyExecution:            "true",
					call.DataTypeExecuteNextMasterOnHangup: "true",
					call.DataTypeAnonymous:                 "false",
				},
				Direction:   call.DirectionOutgoing,
				GroupcallID: uuid.Nil,
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+14155550100",
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
				DialrouteID: uuid.FromStringOrNil("a86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("a86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
						ProviderID: uuid.FromStringOrNil("b213af44-534e-11ed-9a1d-73b0076723b8"),
					},
				},
			},
			expectEndpointDst:  "pjsip/call-out/sip:+821121656521@sip.telnyx.com;transport=udp",
			expectArgs:         "context_type=call,context=call-out,call_id=c1c40962-07fb-11eb-bb82-a3bd16bf1bd9,transport=udp,direction=outgoing",
			expectCodecPresent: true,
			expectCodecValue:   "PCMU",
		},
		{
			name: "provider with empty codecs - VBOUT-CODECS absent from channel variables",

			id:           uuid.FromStringOrNil("d2d40962-07fb-11eb-bb82-a3bd16bf1bd9"),
			customerID:   uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
			flowID:       uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
			activeflowID: uuid.FromStringOrNil("31e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
			masterCallID: uuid.Nil,
			source: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+14155550200",
				TargetName: "test",
			},
			destination: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821121656522",
				TargetName: "test target",
			},
			earlyExecution: true,
			connect:        true,

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
				},
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
			responseRoutes: []rmroute.Route{
				{
					ID:         uuid.FromStringOrNil("b86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
					ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
				},
			},
			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2b095188-2bfe-11ef-a746-7f4de3b06e46"),
				},
			},
			responseUUIDChannel: uuid.FromStringOrNil("f048969e-5de3-11ed-94f5-137ec429b6b6"),
			responseProvider: &rmprovider.Provider{
				ID:       uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
				Hostname: "sip.twilio.com",
				Codecs:   "", // no codecs configured for this provider
			},

			expectProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
			expectCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d2d40962-07fb-11eb-bb82-a3bd16bf1bd9"),
					CustomerID: uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("2b095188-2bfe-11ef-a746-7f4de3b06e46"),
				},
				ChannelID:        "f048969e-5de3-11ed-94f5-137ec429b6b6",
				FlowID:           uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
				ActiveflowID:     uuid.FromStringOrNil("31e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
				Type:             call.TypeFlow,
				ChainedCallIDs:   []uuid.UUID{},
				RecordingIDs:     []uuid.UUID{},
				ExternalMediaIDs: []uuid.UUID{},
				Status:           call.StatusDialing,
				Data: map[call.DataType]string{
					call.DataTypeEarlyExecution:            "true",
					call.DataTypeExecuteNextMasterOnHangup: "true",
					call.DataTypeAnonymous:                 "false",
				},
				Direction:   call.DirectionOutgoing,
				GroupcallID: uuid.Nil,
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+14155550200",
					TargetName: "test",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821121656522",
					TargetName: "test target",
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},
				DialrouteID: uuid.FromStringOrNil("b86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("b86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
						ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
					},
				},
			},
			expectEndpointDst:  "pjsip/call-out/sip:+821121656522@sip.twilio.com;transport=udp",
			expectArgs:         "context_type=call,context=call-out,call_id=d2d40962-07fb-11eb-bb82-a3bd16bf1bd9,transport=udp,direction=outgoing",
			expectCodecPresent: false,
			expectCodecValue:   "",
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
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				utilHandler:           mockUtil,
				reqHandler:            mockReq,
				notifyHandler:         mockNotify,
				db:                    mockDB,
				channelHandler:        mockChannel,
				outboundConfigHandler: mockOutboundConfig,
			}

			ctx := context.Background()

			// outbound config: return a permissive config (KR in whitelist) for tel destination
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(&outboundconfig.OutboundConfig{
				DestinationWhitelist: []string{"kr"},
			}, nil)

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.customerID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().RouteV1DialrouteList(ctx, gomock.Any(), gomock.Any()).Return(tt.responseRoutes, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannel)
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(&cucustomer.Customer{
				ID:                         tt.customerID,
				Status:                     cucustomer.StatusActive,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)

			// source number validation: source belongs to customer as a normal number
			mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), map[nmnumber.Field]any{
				nmnumber.FieldCustomerID: tt.customerID,
				nmnumber.FieldNumber:     tt.source.Target,
				nmnumber.FieldType:       nmnumber.TypeNormal,
				nmnumber.FieldStatus:     nmnumber.StatusActive,
				nmnumber.FieldDeleted:    false,
			}).Return([]nmnumber.Number{{Number: tt.source.Target}}, nil)

			mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.customerID, tt.destination).Return(tt.responseAgent, nil)

			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)
			mockReq.EXPECT().CallV1CallHealth(ctx, tt.expectCall.ID, defaultHealthDelay, 0).Return(nil)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)

			// Capture the channel variables passed to StartChannel so we can assert on them.
			var capturedVariables map[string]string
			mockChannel.EXPECT().StartChannel(
				ctx,
				requesthandler.AsteriskIDCall,
				gomock.Any(),
				tt.expectArgs,
				tt.expectEndpointDst,
				"", "", "",
				gomock.Any(),
			).DoAndReturn(func(_ context.Context, _ string, _ string, _ string, _ string, _, _, _ string, vars map[string]string) (*channel.Channel, error) {
				capturedVariables = vars
				return &channel.Channel{}, nil
			})

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, uuid.Nil, tt.source, tt.destination, tt.earlyExecution, tt.connect, "", nil, nil)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Fatalf("Wrong match. expected non-nil call result")
			}

			// Core assertion: check VBOUT-CODECS presence/value in channel variables.
			val, present := capturedVariables[codecHeaderKey]
			if present != tt.expectCodecPresent {
				t.Errorf("VBOUT-CODECS presence: got %v, want %v. variables=%v", present, tt.expectCodecPresent, capturedVariables)
			}
			if tt.expectCodecPresent && val != tt.expectCodecValue {
				t.Errorf("VBOUT-CODECS value: got %q, want %q", val, tt.expectCodecValue)
			}
		})
	}
}

