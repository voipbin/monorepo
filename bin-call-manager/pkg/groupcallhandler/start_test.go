package groupcallhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	agagent "monorepo/bin-agent-manager/models/agent"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Start_ringall(t *testing.T) {

	tests := []struct {
		name string

		id                uuid.UUID
		customerID        uuid.UUID
		flowID            uuid.UUID
		source            *commonaddress.Address
		destinations      []commonaddress.Address
		masterCallID      uuid.UUID
		masterGroupcallID uuid.UUID
		ringMethod        groupcall.RingMethod
		answerMethod      groupcall.AnswerMethod

		responseUUIDs     []uuid.UUID
		responseAgent     *agagent.Agent
		responseExtension *rmextension.Extension
		responseCall      *call.Call

		expectGroupcall             *groupcall.Groupcall
		expectCallIDs               []uuid.UUID
		expectGroupcallIDs          []uuid.UUID
		expectCallDestinations      []*commonaddress.Address
		expectGroupcallDestinations []*commonaddress.Address
	}{
		{
			name: "call destination only",

			id:         uuid.FromStringOrNil("df3b830c-e468-11ed-adc8-bb725367b1c4"),
			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			flowID:     uuid.FromStringOrNil("40e71e72-bbe7-11ed-9334-a7afb83b403e"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			masterCallID:      uuid.FromStringOrNil("41e86dbc-bbe7-11ed-b8e6-9b0e694bbd6a"),
			masterGroupcallID: uuid.FromStringOrNil("df82ebde-e468-11ed-b9c0-a3de9b27d448"),
			ringMethod:        groupcall.RingMethodRingAll,
			answerMethod:      groupcall.AnswerMethodHangupOthers,

			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("b521af3c-bbe7-11ed-910d-673d428424ab"),
			},
			responseAgent: &agagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("98fcf9ca-2c01-11ef-a404-cbad07804e20"),
					CustomerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
				},
			},

			expectGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("df3b830c-e468-11ed-adc8-bb725367b1c4"),
					CustomerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("98fcf9ca-2c01-11ef-a404-cbad07804e20"),
				},
				Status: groupcall.StatusProgressing,
				FlowID: uuid.FromStringOrNil("40e71e72-bbe7-11ed-9334-a7afb83b403e"),
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
				},
				MasterCallID:      uuid.FromStringOrNil("41e86dbc-bbe7-11ed-b8e6-9b0e694bbd6a"),
				MasterGroupcallID: uuid.FromStringOrNil("df82ebde-e468-11ed-b9c0-a3de9b27d448"),
				RingMethod:        groupcall.RingMethodRingAll,
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("b521af3c-bbe7-11ed-910d-673d428424ab"),
				},
				GroupcallIDs:   []uuid.UUID{},
				CallCount:      1,
				GroupcallCount: 0,
				DialIndex:      0,
			},
			expectCallIDs: []uuid.UUID{
				uuid.FromStringOrNil("b521af3c-bbe7-11ed-910d-673d428424ab"),
			},
			expectCallDestinations: []*commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
		},
		{
			name: "groupcall destination only",

			id:         uuid.FromStringOrNil("0bdd8a4b-6b49-45d9-b4e2-7f0c5e792519"),
			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			flowID:     uuid.FromStringOrNil("ad291cc4-f37a-4438-be8d-53b3c61ca40d"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeAgent,
					Target: "997e5a1a-2c01-11ef-a85e-4713a1f9259c",
				},
			},
			masterCallID:      uuid.FromStringOrNil("41e86dbc-bbe7-11ed-b8e6-9b0e694bbd6a"),
			masterGroupcallID: uuid.FromStringOrNil("df82ebde-e468-11ed-b9c0-a3de9b27d448"),
			ringMethod:        groupcall.RingMethodRingAll,
			answerMethod:      groupcall.AnswerMethodHangupOthers,

			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("c77bbfba-3856-4188-84a3-d735612dcc7d"),
			},
			responseAgent: &agagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("997e5a1a-2c01-11ef-a85e-4713a1f9259c"),
					CustomerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
				},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
				},
			},

			expectGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0bdd8a4b-6b49-45d9-b4e2-7f0c5e792519"),
					CustomerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("997e5a1a-2c01-11ef-a85e-4713a1f9259c"),
				},
				Status: groupcall.StatusProgressing,
				FlowID: uuid.FromStringOrNil("ad291cc4-f37a-4438-be8d-53b3c61ca40d"),
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeAgent,
						Target: "997e5a1a-2c01-11ef-a85e-4713a1f9259c",
					},
				},
				MasterCallID:      uuid.FromStringOrNil("41e86dbc-bbe7-11ed-b8e6-9b0e694bbd6a"),
				MasterGroupcallID: uuid.FromStringOrNil("df82ebde-e468-11ed-b9c0-a3de9b27d448"),
				RingMethod:        groupcall.RingMethodRingAll,
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("c77bbfba-3856-4188-84a3-d735612dcc7d"),
				},
				GroupcallIDs:   []uuid.UUID{},
				CallCount:      1,
				GroupcallCount: 0,
				DialIndex:      0,
			},
			expectCallIDs: []uuid.UUID{
				uuid.FromStringOrNil("c77bbfba-3856-4188-84a3-d735612dcc7d"),
			},
			expectCallDestinations: []*commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
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

			for _, id := range tt.responseUUIDs {
				mockUtil.EXPECT().UUIDCreate().Return(id)
			}

			// getDialAddressesAndRingMethod
			switch tt.destinations[0].Type {
			case commonaddress.TypeAgent:
				mockReq.EXPECT().AgentV1AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
			case commonaddress.TypeExtension:
				mockReq.EXPECT().RegistrarV1ContactGets(ctx, gomock.Any()).Return(tt.responseAgent, nil)
			}

			// getAddressOwner
			if tt.destinations[0].Type == commonaddress.TypeAgent {
				mockReq.EXPECT().AgentV1AgentGet(ctx, tt.responseAgent.ID).Return(tt.responseAgent, nil)
			} else {
				mockReq.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, tt.customerID, tt.destinations[0]).Return(tt.responseAgent, nil)
			}

			// create
			mockDB.EXPECT().GroupcallCreate(ctx, tt.expectGroupcall).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, gomock.Any()).Return(tt.expectGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectGroupcall.CustomerID, groupcall.EventTypeGroupcallCreated, tt.expectGroupcall)

			// create chained call
			for i, destination := range tt.expectCallDestinations {
				mockReq.EXPECT().CallV1CallCreateWithID(ctx, tt.expectCallIDs[i], tt.customerID, tt.flowID, uuid.Nil, tt.masterCallID, tt.source, destination, tt.expectGroupcall.ID, false, false).Return(&call.Call{}, nil)
			}

			// create chained groupcall
			for i, destination := range tt.expectGroupcallDestinations {
				ringMethod := groupcall.RingMethodNone
				if destination.Type == commonaddress.TypeAgent {
					ringMethod = groupcall.RingMethodRingAll
					tmpAgent := &agagent.Agent{
						Identity: commonidentity.Identity{
							CustomerID: tt.customerID,
						},
						RingMethod: agagent.RingMethodRingAll,
					}
					mockReq.EXPECT().AgentV1AgentGet(ctx, gomock.Any()).Return(tmpAgent, nil)
				}
				mockReq.EXPECT().CallV1GroupcallCreate(ctx, tt.expectGroupcallIDs[i], tt.customerID, tt.flowID, *tt.source, gomock.Any(), tt.masterCallID, tt.expectGroupcall.ID, ringMethod, groupcall.AnswerMethodHangupOthers).Return(&groupcall.Groupcall{}, nil)
			}

			res, err := h.Start(ctx, tt.id, tt.customerID, tt.flowID, tt.source, tt.destinations, tt.masterCallID, tt.masterGroupcallID, tt.ringMethod, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(res, tt.expectGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectGroupcall, res)
			}
		})
	}
}

func Test_Start_linear(t *testing.T) {

	tests := []struct {
		name string

		id                uuid.UUID
		customerID        uuid.UUID
		flowID            uuid.UUID
		source            *commonaddress.Address
		destinations      []commonaddress.Address
		masterCallID      uuid.UUID
		masterGroupcallID uuid.UUID
		answerMethod      groupcall.AnswerMethod

		responseUUID      uuid.UUID
		responseGroupcall *groupcall.Groupcall

		expectDialDestination []*commonaddress.Address
	}{
		{
			name: "call destination only",

			id:         uuid.FromStringOrNil("12d4525c-e469-11ed-91fc-2b73eae10785"),
			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			flowID:     uuid.FromStringOrNil("40e71e72-bbe7-11ed-9334-a7afb83b403e"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			masterCallID:      uuid.FromStringOrNil("41e86dbc-bbe7-11ed-b8e6-9b0e694bbd6a"),
			masterGroupcallID: uuid.FromStringOrNil("130d21ea-e469-11ed-b970-2b8dde4c7c94"),
			answerMethod:      groupcall.AnswerMethodHangupOthers,

			responseUUID: uuid.FromStringOrNil("f2997eda-1cea-4ab6-bd98-d2a11d3bf20d"),
			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b521af3c-bbe7-11ed-910d-673d428424ab"),
				},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("edf8f332-bbe8-11ed-a2a7-63ae0390190e"),
				},
			},

			expectDialDestination: []*commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)

			mockDB.EXPECT().GroupcallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, gomock.Any()).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallCreated, tt.responseGroupcall)

			if tt.destinations[0].Type == commonaddress.TypeAgent {
				// todo: need to add the test
			} else {
				mockReq.EXPECT().CallV1CallCreateWithID(ctx, tt.responseUUID, tt.customerID, tt.flowID, uuid.Nil, tt.masterCallID, tt.source, &tt.destinations[0], tt.responseGroupcall.ID, false, false).Return(&call.Call{}, nil)
			}

			res, err := h.Start(ctx, tt.id, tt.customerID, tt.flowID, tt.source, tt.destinations, tt.masterCallID, tt.masterGroupcallID, groupcall.RingMethodLinear, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(res, tt.responseGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}
