package groupcallhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	rmastcontact "monorepo/bin-registrar-manager/models/astcontact"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_dialNextDestination_call(t *testing.T) {

	tests := []struct {
		name string

		groupcall *groupcall.Groupcall

		responseUUID      uuid.UUID
		responseGroupcall *groupcall.Groupcall
		responseCall      *call.Call

		expectDestination *commonaddress.Address
		expectCallIDs     []uuid.UUID
		expectCallCount   int
		expectDialIndex   int
	}{
		{
			name: "normal",

			groupcall: &groupcall.Groupcall{
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("ffc90920-e1fa-11ed-bd02-432621b25d2e"),
				},
				DialIndex: 0,
			},

			responseUUID: uuid.FromStringOrNil("2c5cf172-e1fb-11ed-9c31-cb222c40d531"),

			responseGroupcall: &groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("4b3ad8d2-f7b6-4d8f-868b-364c25c18f6b"),
				CallCount:  0,
				RingMethod: groupcall.RingMethodRingAll,
			},
			responseCall: &call.Call{
				ID: uuid.FromStringOrNil("73529bb2-e233-11ed-9595-1f0b37264b28"),
			},

			expectDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			expectCallIDs: []uuid.UUID{
				uuid.FromStringOrNil("ffc90920-e1fa-11ed-bd02-432621b25d2e"),
				uuid.FromStringOrNil("2c5cf172-e1fb-11ed-9c31-cb222c40d531"),
			},
			expectCallCount: 1,
			expectDialIndex: 1,
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, tt.groupcall.ID, tt.expectCallIDs, tt.expectCallCount, tt.expectDialIndex).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcall.ID).Return(tt.responseGroupcall, nil)

			mockReq.EXPECT().CallV1CallCreateWithID(ctx, tt.responseUUID, tt.groupcall.CustomerID, tt.groupcall.FlowID, uuid.Nil, tt.groupcall.MasterCallID, tt.groupcall.Source, tt.expectDestination, tt.responseGroupcall.ID, false, false).Return(tt.responseCall, nil)

			res, err := h.dialNextDestination(ctx, tt.groupcall)
			if err != nil {
				t.Errorf("wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(tt.responseGroupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_dialNextDestination_groupcall(t *testing.T) {

	tests := []struct {
		name string

		groupcall *groupcall.Groupcall

		responseUUID      uuid.UUID
		responseGroupcall *groupcall.Groupcall
		responseAgent     *amagent.Agent

		expectGroupcallIDs        []uuid.UUID
		expectGroupcallCount      int
		expectGroupcallRingMethod groupcall.RingMethod
		expectDialIndex           int
	}{
		{
			name: "normal",

			groupcall: &groupcall.Groupcall{
				CustomerID: uuid.FromStringOrNil("e591e541-3a4d-4afd-b01e-d27c7db42dea"),
				Source:     &commonaddress.Address{},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
					{
						Type:   commonaddress.TypeAgent,
						Target: "04e3700b-a20b-4f0c-8db7-f58b4ddd4f42",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("ffc90920-e1fa-11ed-bd02-432621b25d2e"),
				},
				GroupcallIDs: []uuid.UUID{},
				DialIndex:    0,
			},

			responseUUID: uuid.FromStringOrNil("2c5cf172-e1fb-11ed-9c31-cb222c40d531"),

			responseGroupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("4b3ad8d2-f7b6-4d8f-868b-364c25c18f6b"),
				CustomerID:   uuid.FromStringOrNil("e591e541-3a4d-4afd-b01e-d27c7db42dea"),
				Source:       &commonaddress.Address{},
				Destinations: []commonaddress.Address{},
				CallCount:    0,
				RingMethod:   groupcall.RingMethodLinear,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("ffc90920-e1fa-11ed-bd02-432621b25d2e"),
				},
				GroupcallIDs: []uuid.UUID{},
				DialIndex:    0,
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
			},
			responseAgent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("04e3700b-a20b-4f0c-8db7-f58b4ddd4f42"),
				CustomerID: uuid.FromStringOrNil("e591e541-3a4d-4afd-b01e-d27c7db42dea"),
				RingMethod: amagent.RingMethodRingAll,
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000010",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000011",
					},
				},
			},

			expectGroupcallIDs: []uuid.UUID{
				uuid.FromStringOrNil("2c5cf172-e1fb-11ed-9c31-cb222c40d531"),
			},
			expectGroupcallCount:      1,
			expectGroupcallRingMethod: groupcall.RingMethodRingAll,
			expectDialIndex:           1,
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

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx, tt.groupcall.ID, tt.expectGroupcallIDs, tt.expectGroupcallCount, tt.expectDialIndex).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcall.ID).Return(tt.responseGroupcall, nil)

			mockReq.EXPECT().CallV1GroupcallCreate(ctx, tt.responseUUID, tt.responseGroupcall.CustomerID, tt.responseGroupcall.FlowID, *tt.responseGroupcall.Source, tt.responseAgent.Addresses, tt.responseGroupcall.MasterCallID, tt.responseGroupcall.ID, tt.expectGroupcallRingMethod, tt.responseGroupcall.AnswerMethod).Return(&groupcall.Groupcall{}, nil)

			res, err := h.dialNextDestination(ctx, tt.groupcall)
			if err != nil {
				t.Errorf("wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(tt.responseGroupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_getDialDestinationsAddressTypeExtension(t *testing.T) {

	tests := []struct {
		name string

		cusotmerID  uuid.UUID
		destination *commonaddress.Address

		responseContacts []rmastcontact.AstContact

		expectRes []commonaddress.Address
	}{
		{
			name: "normal",

			cusotmerID: uuid.FromStringOrNil("20b8c8aa-41cd-438e-8fd3-4bb18b72db67"),
			destination: &commonaddress.Address{
				Type:       commonaddress.TypeExtension,
				TargetName: "test extension",
				Target:     "test-exten",
			},

			responseContacts: []rmastcontact.AstContact{
				{
					URI: "sip:test-exten1@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
				{
					URI: "sip:test-exten2@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
			},

			expectRes: []commonaddress.Address{
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

			mockReq.EXPECT().RegistrarV1ContactGets(ctx, tt.cusotmerID, tt.destination.TargetName).Return(tt.responseContacts, nil)
			res, err := h.getDialDestinationsAddressTypeExtension(ctx, tt.cusotmerID, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getDialDestinationsAddressAndRingMethodTypeAgent(t *testing.T) {

	tests := []struct {
		name string

		cusotmerID  uuid.UUID
		destination *commonaddress.Address

		responseAgent *amagent.Agent

		expectResAddresses  []commonaddress.Address
		expectResRingMethod groupcall.RingMethod
	}{
		{
			name: "normal",

			cusotmerID: uuid.FromStringOrNil("1a7c277a-d3e5-448c-be17-78b80bc19844"),
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "a47ddec7-bf14-4030-9303-a660481dad8f",
			},

			responseAgent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("a47ddec7-bf14-4030-9303-a660481dad8f"),
				CustomerID: uuid.FromStringOrNil("1a7c277a-d3e5-448c-be17-78b80bc19844"),
				RingMethod: amagent.RingMethodRingAll,
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeSIP,
						Target: "sip:test@test.com",
					},
					{
						Type:   commonaddress.TypeExtension,
						Target: "test-exten",
					},
				},
			},

			expectResAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeSIP,
					Target: "sip:test@test.com",
				},
				{
					Type:   commonaddress.TypeExtension,
					Target: "test-exten",
				},
			},
			expectResRingMethod: groupcall.RingMethodRingAll,
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

			mockReq.EXPECT().AgentV1AgentGet(ctx, uuid.FromStringOrNil(tt.destination.Target)).Return(tt.responseAgent, nil)
			resAddresses, resRingMethod, err := h.getDialDestinationsAddressAndRingMethodTypeAgent(ctx, tt.cusotmerID, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(tt.expectResAddresses, resAddresses) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResAddresses, resAddresses)
			}
			if !reflect.DeepEqual(tt.expectResRingMethod, resRingMethod) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResRingMethod, resRingMethod)
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
		expectResOwnerType groupcall.OwnerType
		expectResOwnerID   uuid.UUID
	}{
		{
			name: "normal - address type is agent",

			customerID: uuid.FromStringOrNil("8e1a8f84-2fd7-11ef-a27d-ab76183e2c6b"),
			address: &commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "8e3b890a-2fd7-11ef-b442-133f59be8b36",
			},

			responseAgent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("8e3b890a-2fd7-11ef-b442-133f59be8b36"),
				CustomerID: uuid.FromStringOrNil("8e1a8f84-2fd7-11ef-a27d-ab76183e2c6b"),
			},

			expectAgentID:      uuid.FromStringOrNil("8e3b890a-2fd7-11ef-b442-133f59be8b36"),
			expectResOwnerType: groupcall.OwnerTypeAgent,
			expectResOwnerID:   uuid.FromStringOrNil("8e3b890a-2fd7-11ef-b442-133f59be8b36"),
		},
		{
			name: "normal - address type is not agent",

			customerID: uuid.FromStringOrNil("8e5ddef6-2fd7-11ef-82a7-235a64789cf8"),
			address: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+123456789",
			},

			responseAgent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("8e7f4af0-2fd7-11ef-9c3a-53f59238b991"),
				CustomerID: uuid.FromStringOrNil("8e5ddef6-2fd7-11ef-82a7-235a64789cf8"),
			},

			expectResOwnerType: groupcall.OwnerTypeAgent,
			expectResOwnerID:   uuid.FromStringOrNil("8e7f4af0-2fd7-11ef-9c3a-53f59238b991"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &groupcallHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
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
