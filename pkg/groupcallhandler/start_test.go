package groupcallhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	rmastcontact "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_getDialDestinations_getDialDestinationsAddressTypeAgent(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		destinations []commonaddress.Address

		responseAgents []*amagent.Agent

		expectAgentID uuid.UUID
		expectRes     []*commonaddress.Address
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			destinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeAgent,
					Target:     "081fd090-e4ba-401a-97a0-d36dd1f12f75",
					TargetName: "test agent",
				},
			},

			responseAgents: []*amagent.Agent{
				{
					ID:         uuid.FromStringOrNil("081fd090-e4ba-401a-97a0-d36dd1f12f75"),
					CustomerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
					Addresses: []commonaddress.Address{
						{
							Type:   commonaddress.TypeTel,
							Target: "+821100000001",
						},
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			for i, destination := range tt.destinations {
				agentID := uuid.FromStringOrNil(destination.Target)
				mockReq.EXPECT().AgentV1AgentGet(ctx, agentID).Return(tt.responseAgents[i], nil)
			}

			res, err := h.getDialDestinations(ctx, tt.customerID, tt.destinations)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getDialDestinations_getdialDestinationsAddressTypeEndpoint(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		destinations []commonaddress.Address

		responseExntensions []*rmextension.Extension
		responseContacts    []rmastcontact.AstContact

		expectRes []*commonaddress.Address
	}{
		{
			name: "1 contact",

			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			destinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEndpoint,
					Target:     "test-exten@test-domain",
					TargetName: "test extension",
				},
			},

			responseExntensions: []*rmextension.Extension{
				{
					ID:         uuid.FromStringOrNil("382bee00-b5ef-11ed-a3d9-f7ee8324e815"),
					CustomerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
				},
			},
			responseContacts: []rmastcontact.AstContact{
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
			destinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEndpoint,
					Target:     "test-exten@test-domain",
					TargetName: "test extension",
				},
			},

			responseExntensions: []*rmextension.Extension{
				{
					ID:         uuid.FromStringOrNil("ea46f164-b5c4-11ed-823e-47dfafd09c7b"),
					CustomerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
				},
			},
			responseContacts: []rmastcontact.AstContact{
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			for i, destination := range tt.destinations {
				mockReq.EXPECT().RegistrarV1ExtensionGetByEndpoint(ctx, destination.Target).Return(tt.responseExntensions[i], nil)
				mockReq.EXPECT().RegistrarV1ContactGets(ctx, destination.Target).Return(tt.responseContacts, nil)
			}

			res, err := h.getDialDestinations(ctx, tt.customerID, tt.destinations)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Start(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		source       *commonaddress.Address
		destinations []commonaddress.Address
		flowID       uuid.UUID
		masterCallID uuid.UUID
		ringMethod   groupcall.RingMethod
		answerMethod groupcall.AnswerMethod

		responseGroupcall *groupcall.Groupcall

		expectDialDestination []*commonaddress.Address
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},
			flowID:       uuid.FromStringOrNil("40e71e72-bbe7-11ed-9334-a7afb83b403e"),
			masterCallID: uuid.FromStringOrNil("41e86dbc-bbe7-11ed-b8e6-9b0e694bbd6a"),
			ringMethod:   groupcall.RingMethodRingAll,
			answerMethod: groupcall.AnswerMethodHangupOthers,

			responseGroupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("b521af3c-bbe7-11ed-910d-673d428424ab"),
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("edf8f332-bbe8-11ed-a2a7-63ae0390190e"),
				},
			},

			expectDialDestination: []*commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
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

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			for i := range tt.expectDialDestination {
				mockUtil.EXPECT().CreateUUID().Return(tt.responseGroupcall.CallIDs[i])
			}

			mockUtil.EXPECT().CreateUUID().Return(tt.responseGroupcall.ID)
			mockDB.EXPECT().GroupcallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, gomock.Any()).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupcall.EventTypeGroupcallCreated, gomock.Any())

			for i, dialDestination := range tt.expectDialDestination {
				mockReq.EXPECT().CallV1CallCreateWithID(ctx, tt.responseGroupcall.CallIDs[i], tt.customerID, tt.flowID, uuid.Nil, tt.masterCallID, tt.source, dialDestination, tt.responseGroupcall.ID, false, false).Return(&call.Call{}, nil)
			}

			res, err := h.Start(ctx, tt.customerID, tt.source, tt.destinations, tt.flowID, tt.masterCallID, tt.ringMethod, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}
