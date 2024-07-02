package agenthandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_dbGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string
		filters    map[string]string

		responseAgents []*agent.Agent
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			size:       10,
			token:      "2021-11-23 17:55:39.712000",
			filters: map[string]string{
				"deleted": "false",
			},

			responseAgents: []*agent.Agent{},
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

			mockDB.EXPECT().AgentGets(gomock.Any(), tt.size, tt.token, tt.filters).Return(tt.responseAgents, nil)
			_, err := h.dbGets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_dbLogin(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		responseAgents *agent.Agent
	}{
		{
			name: "normal",

			username: "ee0b3c8c-298d-11ee-ab0f-3ba8c0a1b163",
			password: "ee413d32-298d-11ee-ac24-0fe74d810696",

			responseAgents: &agent.Agent{
				PasswordHash: "$2a$12$cCNXtIMoEuEs0fKr.Ij8fuim1ZXPTtzhkBSLxmTdlC.CNqQiZSXw2",
			},
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

			mockDB.EXPECT().AgentGetByUsername(ctx, tt.username).Return(tt.responseAgents, nil)
			_, err := h.dbLogin(ctx, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_dbUpdateInfo(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		agentName  string
		detail     string
		ringMethod agent.RingMethod

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("e9dddd94-298e-11ee-9182-c37003ff92d7"),
			agentName:  "update name",
			detail:     "update detail",
			ringMethod: agent.RingMethodRingAll,

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9dddd94-298e-11ee-9182-c37003ff92d7"),
				},
			},
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

			mockDB.EXPECT().AgentSetBasicInfo(ctx, tt.id, tt.agentName, tt.detail, tt.ringMethod).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, tt.responseAgent)
			res, err := h.dbUpdateInfo(ctx, tt.id, tt.agentName, tt.detail, tt.ringMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAgent, res)
			}
		})
	}
}

func Test_dbUpdatePassword(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		password string

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id:       uuid.FromStringOrNil("901d2638-298f-11ee-a14a-639e3e00089b"),
			password: "904ae97e-298f-11ee-8527-1bdc00790371",

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("901d2638-298f-11ee-a14a-639e3e00089b"),
				},
			},
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

			mockDB.EXPECT().AgentSetPasswordHash(ctx, tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, tt.responseAgent)

			res, err := h.dbUpdatePassword(ctx, tt.id, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAgent, res)
			}
		})
	}
}

func Test_dbUpdatePermission(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		permission agent.Permission

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("411f1f22-2990-11ee-a246-c366dbfd41ec"),
			permission: agent.PermissionNone,

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("411f1f22-2990-11ee-a246-c366dbfd41ec"),
				},
			},
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

			mockDB.EXPECT().AgentSetPermission(ctx, tt.id, tt.permission).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, tt.responseAgent)

			res, err := h.dbUpdatePermission(ctx, tt.id, tt.permission)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAgent, res)
			}
		})
	}
}

func Test_dbUpdateTagIDs(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		tagIDs []uuid.UUID

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("741556bc-2990-11ee-bba5-f3d4e2e1d710"),
			tagIDs: []uuid.UUID{
				uuid.FromStringOrNil("743eae90-2990-11ee-92bd-03b99d13221a"),
				uuid.FromStringOrNil("74758ec4-2990-11ee-acc8-77a5db8f6091"),
			},

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("741556bc-2990-11ee-bba5-f3d4e2e1d710"),
				},
			},
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

			mockDB.EXPECT().AgentSetTagIDs(ctx, tt.id, tt.tagIDs).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, tt.responseAgent)

			res, err := h.dbUpdateTagIDs(ctx, tt.id, tt.tagIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAgent, res)
			}
		})
	}
}

func Test_dbUpdateAddresses(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		addresses []commonaddress.Address

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("741556bc-2990-11ee-bba5-f3d4e2e1d710"),
			addresses: []commonaddress.Address{
				{
					Target: "+821100000001",
				},
				{
					Target: "+821100000002",
				},
			},

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("741556bc-2990-11ee-bba5-f3d4e2e1d710"),
				},
			},
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

			mockDB.EXPECT().AgentSetAddresses(ctx, tt.id, tt.addresses).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishEvent(ctx, agent.EventTypeAgentUpdated, tt.responseAgent)

			res, err := h.dbUpdateAddresses(ctx, tt.id, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAgent, res)
			}
		})
	}
}

func Test_dbUpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status agent.Status

		responseAgent *agent.Agent
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("c9f84f58-2990-11ee-8594-5f437170c924"),
			status: agent.StatusBusy,

			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c9f84f58-2990-11ee-8594-5f437170c924"),
				},
			},
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

			mockDB.EXPECT().AgentSetStatus(ctx, tt.id, tt.status).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentStatusUpdated, tt.responseAgent)

			res, err := h.dbUpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAgent) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAgent, res)
			}
		})
	}
}
