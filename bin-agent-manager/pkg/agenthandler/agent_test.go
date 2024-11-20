package agenthandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		result []*agent.Agent
	}{
		{
			"normal",

			10,
			"2021-11-23 17:55:39.712000",
			map[string]string{
				"deleted": "false",
			},

			[]*agent.Agent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentGets(gomock.Any(), tt.size, tt.token, tt.filters).Return(tt.result, nil)
			_, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		username   string
		password   string
		agentName  string
		detail     string
		ringMethod agent.RingMethod
		permission agent.Permission
		tags       []uuid.UUID
		addresses  []commonaddress.Address

		responseUUID uuid.UUID
		expectRes    *agent.Agent
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			username:   "test1@voipbin.net",
			password:   "test1password",
			agentName:  "test1 name",
			detail:     "test1 detail",
			ringMethod: agent.RingMethodRingAll,
			permission: agent.PermissionNone,
			tags:       []uuid.UUID{},
			addresses:  []commonaddress.Address{},

			responseUUID: uuid.FromStringOrNil("ac810dc4-298c-11ee-984c-ebb7811c4114"),
			expectRes: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("89a42670-4c4c-11ec-86ed-9b96390f7668"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username:   "test1@voipbin.net",
				Name:       "test1 name",
				Detail:     "test1 detail",
				Permission: agent.PermissionNone,
				TagIDs:     []uuid.UUID{},
				Addresses:  []commonaddress.Address{},
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

			h := &agentHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentGetByUsername(ctx, tt.username).Return(nil, fmt.Errorf("not found"))
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().AgentCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, gomock.Any()).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectRes.CustomerID, agent.EventTypeAgentCreated, tt.expectRes)

			res, err := h.Create(ctx, tt.customerID, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tags, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		responseAgent *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username:   "test2",
				Name:       "test2 name",
				Detail:     "test2 detail",
				Permission: agent.PermissionNone,
				TagIDs:     []uuid.UUID{},
				Addresses:  []commonaddress.Address{},
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

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// is only admin
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)

			mockDB.EXPECT().AgentDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentDeleted, tt.responseAgent)

			_, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_deleteForce(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		responseAgent *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("073fb108-e746-11ee-8a34-033accc28b49"),

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("073fb108-e746-11ee-8a34-033accc28b49"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username:   "test2",
				Name:       "test2 name",
				Detail:     "test2 detail",
				Permission: agent.PermissionNone,
				TagIDs:     []uuid.UUID{},
				Addresses:  []commonaddress.Address{},
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

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentDeleted, tt.responseAgent)

			_, err := h.deleteForce(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_UpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		status        agent.Status
		responseAgent *agent.Agent
	}{
		{
			"available",

			uuid.FromStringOrNil("1f7e03de-79a5-11ec-ac0a-4f99eb1b36e8"),
			agent.StatusAvailable,

			&agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1f7e03de-79a5-11ec-ac0a-4f99eb1b36e8"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				},
				Username:   "test2",
				Name:       "test2 name",
				Detail:     "test2 detail",
				Status:     agent.StatusAvailable,
				Permission: agent.PermissionNone,
				TagIDs:     []uuid.UUID{},
				Addresses:  []commonaddress.Address{},
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

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentSetStatus(ctx, tt.id, tt.status).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentStatusUpdated, tt.responseAgent)

			_, err := h.UpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_GetByCustomerIDAndAddress(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		address    *commonaddress.Address

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("d3aab1b0-2d88-11ef-ba6a-afc97b6c3b32"),
			address: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "d4116ac2-2d88-11ef-9795-8393fdf24d82",
			},

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d3eaf9d2-2d88-11ef-9997-7bea6cbbf856"),
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

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentGetByCustomerIDAndAddress(ctx, tt.customerID, tt.address).Return(tt.responseAgent, nil)

			res, err := h.GetByCustomerIDAndAddress(ctx, tt.customerID, tt.address)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", res, tt.responseAgent)
			}
		})
	}
}

func Test_isOnlyAdmin(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAgent  *agent.Agent
		responseAgents []*agent.Agent

		expectFilters map[string]string
		expectRes     bool
	}{
		{
			name: "agent is the only admin with other agents",

			id: uuid.FromStringOrNil("9156fd04-e73e-11ee-b987-bbd3786d50b0"),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9156fd04-e73e-11ee-b987-bbd3786d50b0"),
					CustomerID: uuid.FromStringOrNil("ae51a166-e73e-11ee-92dd-07437d91f85c"),
				},
				Permission: agent.PermissionCustomerAdmin,
			},
			responseAgents: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("9156fd04-e73e-11ee-b987-bbd3786d50b0"),
						CustomerID: uuid.FromStringOrNil("ae51a166-e73e-11ee-92dd-07437d91f85c"),
					},
					Permission: agent.PermissionCustomerAdmin,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("9156fd04-e73e-11ee-b987-bbd3786d50b0"),
						CustomerID: uuid.FromStringOrNil("ae51a166-e73e-11ee-92dd-07437d91f85c"),
					},
					Permission: agent.PermissionCustomerAgent,
				},
			},

			expectFilters: map[string]string{
				"customer_id": "ae51a166-e73e-11ee-92dd-07437d91f85c",
				"deleted":     "false",
			},
			expectRes: true,
		},
		{
			name: "agent is the only admin",

			id: uuid.FromStringOrNil("062fb072-e743-11ee-9006-73486af008af"),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("062fb072-e743-11ee-9006-73486af008af"),
					CustomerID: uuid.FromStringOrNil("ae51a166-e73e-11ee-92dd-07437d91f85c"),
				},
				Permission: agent.PermissionCustomerAdmin,
			},
			responseAgents: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("062fb072-e743-11ee-9006-73486af008af"),
						CustomerID: uuid.FromStringOrNil("ae51a166-e73e-11ee-92dd-07437d91f85c"),
					},
					Permission: agent.PermissionCustomerAdmin,
				},
			},

			expectFilters: map[string]string{
				"customer_id": "ae51a166-e73e-11ee-92dd-07437d91f85c",
				"deleted":     "false",
			},
			expectRes: true,
		},
		{
			name: "agent is not the only admin",

			id: uuid.FromStringOrNil("3714d7a8-e743-11ee-a810-f759c74aeb9c"),

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3714d7a8-e743-11ee-a810-f759c74aeb9c"),
					CustomerID: uuid.FromStringOrNil("ae51a166-e73e-11ee-92dd-07437d91f85c"),
				},
				Permission: agent.PermissionCustomerAdmin,
			},
			responseAgents: []*agent.Agent{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3714d7a8-e743-11ee-a810-f759c74aeb9c"),
						CustomerID: uuid.FromStringOrNil("ae51a166-e73e-11ee-92dd-07437d91f85c"),
					},
					Permission: agent.PermissionCustomerAdmin,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("37b3b29c-e743-11ee-a40d-1b23ea7c53e1"),
						CustomerID: uuid.FromStringOrNil("ae51a166-e73e-11ee-92dd-07437d91f85c"),
					},
					Permission: agent.PermissionCustomerAdmin,
				},
			},

			expectFilters: map[string]string{
				"customer_id": "ae51a166-e73e-11ee-92dd-07437d91f85c",
				"deleted":     "false",
			},
			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockDB.EXPECT().AgentGets(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseAgents, nil)

			res, err := h.isOnlyAdmin(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_UpdateAddresses(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		addresses []commonaddress.Address

		responseAgent     *agent.Agent
		responseExtension *rmextension.Extension
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6"),
			addresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeExtension,
					Target: "49b41028-2d8d-11ef-b38d-27dd55f2bb71",
				},
			},

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6"),
					CustomerID: uuid.FromStringOrNil("49d90a72-2d8d-11ef-b208-fb6caaa88ae9"),
				},
			},
			responseExtension: &rmextension.Extension{
				ID:         uuid.FromStringOrNil("49b41028-2d8d-11ef-b38d-27dd55f2bb71"),
				CustomerID: uuid.FromStringOrNil("49d90a72-2d8d-11ef-b208-fb6caaa88ae9"),
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

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			for _, addr := range tt.addresses {
				switch addr.Type {
				case commonaddress.TypeExtension:
					mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, uuid.FromStringOrNil(addr.Target)).Return(tt.responseExtension, nil)
				}

				mockDB.EXPECT().AgentGetByCustomerIDAndAddress(ctx, tt.responseAgent.CustomerID, &addr).Return(nil, nil)
			}
			mockDB.EXPECT().AgentSetAddresses(ctx, tt.id, tt.addresses).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, tt.responseAgent)

			res, err := h.UpdateAddresses(ctx, tt.id, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", res, tt.responseAgent)
			}
		})
	}
}
