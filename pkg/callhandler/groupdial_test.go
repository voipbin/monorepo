package callhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	rmastcontact "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_createGroupDial(t *testing.T) {

	tests := []struct {
		name string

		customerID                uuid.UUID
		flowID                    uuid.UUID
		masterCallID              uuid.UUID
		source                    *commonaddress.Address
		destination               *commonaddress.Address
		earlyExecution            bool
		executeNextMasterOnHangup bool

		responseExntension *rmextension.Extension
		responseContacts   []*rmastcontact.AstContact
		responseUUID       uuid.UUID
		responseCalls      []*call.Call

		expectGroupDial *groupdial.Groupdial
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			flowID:       uuid.FromStringOrNil("e9ebb18c-b5c4-11ed-9775-cf1b5f3ac127"),
			masterCallID: uuid.Nil,
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeEndpoint,
				Target: "test-exten@test-domain",
			},
			earlyExecution:            false,
			executeNextMasterOnHangup: false,

			responseExntension: &rmextension.Extension{
				ID:         uuid.FromStringOrNil("ea46f164-b5c4-11ed-823e-47dfafd09c7b"),
				CustomerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			},
			responseContacts: []*rmastcontact.AstContact{
				{
					URI: "sip:test-exten1@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
				{
					URI: "sip:test-exten2@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
			},
			responseUUID: uuid.FromStringOrNil("08701bca-b5e8-11ed-9257-4bee6cbc72bf"),
			responseCalls: []*call.Call{
				{
					ID: uuid.FromStringOrNil("a62ac2ae-b5eb-11ed-9607-fff199830675"),
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeSIP,
						Target: "sip:test-exten1@211.200.20.28:53941;transport=udp;alias=211.200.20.28~53941~1",
					},
				},
				{
					ID: uuid.FromStringOrNil("a65415be-b5eb-11ed-86f5-83763d30dbf1"),
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeSIP,
						Target: "sip:test-exten2@211.200.20.28:53941;transport=udp;alias=211.200.20.28~53941~1",
					},
				},
			},

			expectGroupDial: &groupdial.Groupdial{
				ID:         uuid.FromStringOrNil("08701bca-b5e8-11ed-9257-4bee6cbc72bf"),
				CustomerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test-exten@test-domain",
				},
				RingMethod:   groupdial.RingMethodRingAll,
				AnswerMethod: groupdial.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("a62ac2ae-b5eb-11ed-9607-fff199830675"),
					uuid.FromStringOrNil("a65415be-b5eb-11ed-86f5-83763d30dbf1"),
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

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			// getDestinationsAddressTypeEndpoint
			mockReq.EXPECT().RegistrarV1ExtensionGetByEndpoint(ctx, tt.destination.Target).Return(tt.responseExntension, nil)
			mockReq.EXPECT().RegistrarV1ContactGets(ctx, tt.destination.Target).Return(tt.responseContacts, nil)

			for i := range tt.responseContacts {
				// CreateCallOutgoing
				mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
				mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, gomock.Any(), tt.flowID, fmactiveflow.ReferenceTypeCall, gomock.Any()).Return(&fmactiveflow.Activeflow{}, nil)

				mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
				mockDB.EXPECT().CallCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().CallGet(ctx, gomock.Any()).Return(tt.responseCalls[i], nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallCreated, gomock.Any())
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)

				mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, tt.responseCalls[i].ChannelID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)
			}
			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().GroupdialCreate(ctx, tt.expectGroupDial).Return(nil)
			mockDB.EXPECT().GroupdialGet(ctx, tt.expectGroupDial.ID).Return(tt.expectGroupDial, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupdial.EventTypeGroupdialCreated, tt.expectGroupDial)

			res, err := h.createGroupdial(ctx, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destination, tt.earlyExecution, tt.executeNextMasterOnHangup)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectGroupDial) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectGroupDial, res)
			}
		})
	}
}

func Test_getDestinationsAddressTypeEndpoint(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		destination *commonaddress.Address

		responseExntension *rmextension.Extension
		responseContacts   []*rmastcontact.AstContact

		expectRes []*commonaddress.Address
	}{
		{
			name: "1 contact",

			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			destination: &commonaddress.Address{
				Type:       commonaddress.TypeEndpoint,
				Target:     "test-exten@test-domain",
				TargetName: "test extension",
			},

			responseExntension: &rmextension.Extension{
				ID:         uuid.FromStringOrNil("382bee00-b5ef-11ed-a3d9-f7ee8324e815"),
				CustomerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			},
			responseContacts: []*rmastcontact.AstContact{
				{
					URI: "sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22",
				},
			},

			expectRes: []*commonaddress.Address{
				{
					Type:       commonaddress.TypeSIP,
					TargetName: "test extension",
					Target:     "sip:test11@211.178.226.108:35551;transport=UDP;rinstance=8a1f981a77f30a22",
				},
			},
		},
		{
			name: "2 contacts",

			customerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			destination: &commonaddress.Address{
				Type:       commonaddress.TypeEndpoint,
				Target:     "test-exten@test-domain",
				TargetName: "test extension",
			},

			responseExntension: &rmextension.Extension{
				ID:         uuid.FromStringOrNil("ea46f164-b5c4-11ed-823e-47dfafd09c7b"),
				CustomerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			},
			responseContacts: []*rmastcontact.AstContact{
				{
					URI: "sip:test-exten1@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
				{
					URI: "sip:test-exten2@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
			},

			expectRes: []*commonaddress.Address{
				{
					Type:       commonaddress.TypeSIP,
					TargetName: "test extension",
					Target:     "sip:test-exten1@211.200.20.28:53941;transport=udp;alias=211.200.20.28~53941~1",
				},
				{
					Type:       commonaddress.TypeSIP,
					TargetName: "test extension",
					Target:     "sip:test-exten2@211.200.20.28:53941;transport=udp;alias=211.200.20.28~53941~1",
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ExtensionGetByEndpoint(ctx, tt.destination.Target).Return(tt.responseExntension, nil)
			mockReq.EXPECT().RegistrarV1ContactGets(ctx, tt.destination.Target).Return(tt.responseContacts, nil)

			res, err := h.getDestinationsAddressTypeEndpoint(ctx, tt.customerID, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_answerGroupdial(t *testing.T) {

	tests := []struct {
		name string

		groupdialID  uuid.UUID
		answercallID uuid.UUID

		responseGroupDial *groupdial.Groupdial
	}{
		{
			name: "normal",

			groupdialID:  uuid.FromStringOrNil("d3391861-292d-4ed8-b03a-7b455e57b17b"),
			answercallID: uuid.FromStringOrNil("1f142f05-c169-4caa-a6b2-42d224ec6ca5"),

			responseGroupDial: &groupdial.Groupdial{
				ID:           uuid.FromStringOrNil("d3391861-292d-4ed8-b03a-7b455e57b17b"),
				AnswerMethod: groupdial.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("f3a4b38b-4781-4db7-b74a-5958b2851225"),
					uuid.FromStringOrNil("0de4689f-96d5-448d-be00-11c196163756"),
					uuid.FromStringOrNil("1f142f05-c169-4caa-a6b2-42d224ec6ca5"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupdialGet(ctx, tt.groupdialID).Return(tt.responseGroupDial, nil)
			mockDB.EXPECT().GroupdialGet(ctx, tt.groupdialID).Return(tt.responseGroupDial, nil)
			updateGroupDial := *tt.responseGroupDial
			updateGroupDial.AnswerCallID = tt.answercallID
			mockDB.EXPECT().GroupdialUpdate(ctx, &updateGroupDial).Return(nil)
			mockDB.EXPECT().GroupdialGet(ctx, tt.groupdialID).Return(&updateGroupDial, nil)

			for _, callID := range tt.responseGroupDial.CallIDs {

				if callID == tt.answercallID {
					continue
				}

				// HangingUp. just return the error cause it's too long write the test code here.
				mockDB.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf(""))
			}

			if errAnswer := h.answerGroupdial(ctx, tt.groupdialID, tt.answercallID); errAnswer != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errAnswer)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_getDestinationsAddressTypeAgent(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		destination *commonaddress.Address

		responseAgent *amagent.Agent

		expectAgentID uuid.UUID
		expectRes     []*commonaddress.Address
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			destination: &commonaddress.Address{
				Type:       commonaddress.TypeEndpoint,
				Target:     "081fd090-e4ba-401a-97a0-d36dd1f12f75",
				TargetName: "test agent",
			},

			responseAgent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("081fd090-e4ba-401a-97a0-d36dd1f12f75"),
				CustomerID: uuid.FromStringOrNil("239ec41e-7649-4a17-99a4-6729b56f64ac"),
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
				},
			},

			expectAgentID: uuid.FromStringOrNil("081fd090-e4ba-401a-97a0-d36dd1f12f75"),
			expectRes: []*commonaddress.Address{
				{
					Type:       commonaddress.TypeTel,
					TargetName: "test agent",
					Target:     "+821100000001",
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.expectAgentID).Return(tt.responseAgent, nil)

			res, err := h.getDestinationsAddressTypeAgent(ctx, tt.customerID, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
