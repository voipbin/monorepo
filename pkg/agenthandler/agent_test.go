package agenthandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
)

func Test_AgentGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string
		result     []*agent.Agent
	}{
		{
			"normal1",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			10,
			"2021-11-23 17:55:39.712000",
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
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentGets(gomock.Any(), tt.customerID, tt.size, tt.token).Return(tt.result, nil)
			_, err := h.AgentGets(ctx, tt.customerID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_AgentGetsByTags(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		tags       []uuid.UUID

		result    []*agent.Agent
		expectRes []*agent.Agent
	}{
		{
			"normal",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
			},

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("c47a762e-4c87-11ec-b1d8-531dbb4ebcd2"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
					},
				},
			},
			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("c47a762e-4c87-11ec-b1d8-531dbb4ebcd2"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
					},
				},
			},
		},
		{
			"has 2 agents, 1 selected",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
			},

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("5c61f98a-4c88-11ec-9181-43fb8e090ace"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("5c7cf794-4c88-11ec-a55d-b3af0e75c8e1"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs:     []uuid.UUID{},
				},
			},
			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("5c61f98a-4c88-11ec-9181-43fb8e090ace"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
					},
				},
			},
		},
		{
			"has 2 agents, all selected",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
			},

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("7f1d18e2-4c88-11ec-9f6b-4fad140d455c"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("7f3bf4ba-4c88-11ec-ab26-675037d57999"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
			},
			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("7f1d18e2-4c88-11ec-9f6b-4fad140d455c"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("7f3bf4ba-4c88-11ec-ab26-675037d57999"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
			},
		},
		{
			"has 2 agents, none selected",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("9f7746e4-4c88-11ec-9c3a-6b0e38bbc60f"),
			},

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("9f9c03b2-4c88-11ec-ac69-7b00edc54e08"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("9ffe117e-4c88-11ec-9188-4b98b647fe1d"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("9fd03d44-4c88-11ec-9ebe-3fc386a2a1e6"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a02c0a48-4c88-11ec-99da-bb9592c80bf8"),
					},
				},
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
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentGets(gomock.Any(), tt.customerID, uint64(maxAgentCount), gomock.Any()).Return(tt.result, nil)
			res, err := h.AgentGetsByTagIDs(ctx, tt.customerID, tt.tags)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wront match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentGetsByTagIDsAndStatus(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		tags       []uuid.UUID
		status     agent.Status

		result    []*agent.Agent
		expectRes []*agent.Agent
	}{
		{
			"normal",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("c47a762e-4c87-11ec-b1d8-531dbb4ebcd2"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
					},
				},
			},
			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("c47a762e-4c87-11ec-b1d8-531dbb4ebcd2"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
					},
				},
			},
		},
		{
			"has 2 agents, 1 selected",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("5c61f98a-4c88-11ec-9181-43fb8e090ace"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("5c7cf794-4c88-11ec-a55d-b3af0e75c8e1"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs:     []uuid.UUID{},
				},
			},
			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("5c61f98a-4c88-11ec-9181-43fb8e090ace"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
					},
				},
			},
		},
		{
			"has 2 agents, all selected",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("7f1d18e2-4c88-11ec-9f6b-4fad140d455c"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("7f3bf4ba-4c88-11ec-ab26-675037d57999"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
			},
			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("7f1d18e2-4c88-11ec-9f6b-4fad140d455c"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("7f3bf4ba-4c88-11ec-ab26-675037d57999"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
			},
		},
		{
			"has 2 agents, none selected",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("9f7746e4-4c88-11ec-9c3a-6b0e38bbc60f"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("9f9c03b2-4c88-11ec-ac69-7b00edc54e08"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("9ffe117e-4c88-11ec-9188-4b98b647fe1d"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("9fd03d44-4c88-11ec-9ebe-3fc386a2a1e6"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a02c0a48-4c88-11ec-99da-bb9592c80bf8"),
					},
				},
			},
			[]*agent.Agent{},
		},
		{
			"has 2 agents, none selected by wrong status",
			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			[]uuid.UUID{
				uuid.FromStringOrNil("9f7746e4-4c88-11ec-9c3a-6b0e38bbc60f"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:         uuid.FromStringOrNil("9f9c03b2-4c88-11ec-ac69-7b00edc54e08"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusOffline,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("9ffe117e-4c88-11ec-9188-4b98b647fe1d"),
					},
				},
				{
					ID:         uuid.FromStringOrNil("9fd03d44-4c88-11ec-9ebe-3fc386a2a1e6"),
					CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
					Status:     agent.StatusAway,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a02c0a48-4c88-11ec-99da-bb9592c80bf8"),
					},
				},
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
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentGets(gomock.Any(), tt.customerID, uint64(maxAgentCount), gomock.Any()).Return(tt.result, nil)
			res, err := h.AgentGetsByTagIDsAndStatus(ctx, tt.customerID, tt.tags, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wront match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentCreate(t *testing.T) {

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

		expectRes *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
			"test1",
			"test1password",
			"test1 name",
			"test1 detail",
			agent.RingMethodRingAll,
			agent.PermissionNone,
			[]uuid.UUID{},
			[]commonaddress.Address{},

			&agent.Agent{
				ID:         uuid.FromStringOrNil("89a42670-4c4c-11ec-86ed-9b96390f7668"),
				CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				Username:   "test1",
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

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentGetByUsername(gomock.Any(), tt.customerID, tt.username).Return(nil, fmt.Errorf("not found"))
			mockDB.EXPECT().AgentCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().AgentGet(gomock.Any(), gomock.Any()).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectRes.CustomerID, agent.EventTypeAgentCreated, tt.expectRes)

			res, err := h.AgentCreate(ctx, tt.customerID, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tags, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}

			res.ID = uuid.Nil
			res.PasswordHash = ""
			res.TMCreate = ""
			res.TMUpdate = ""
			res.TMDelete = ""

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentDelete(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		responseAgent *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),

			&agent.Agent{
				ID:         uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83"),
				CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
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
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentDeleted, tt.responseAgent)

			_, err := h.AgentDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_AgentUpdateStatus(t *testing.T) {

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
				ID:         uuid.FromStringOrNil("1f7e03de-79a5-11ec-ac0a-4f99eb1b36e8"),
				CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
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
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentSetStatus(ctx, tt.id, tt.status).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, tt.id).Return(tt.responseAgent, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAgent.CustomerID, agent.EventTypeAgentStatusUpdated, tt.responseAgent)

			_, err := h.AgentUpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_AgentDial(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		source       *commonaddress.Address
		flowID       uuid.UUID
		masterCallID uuid.UUID

		agent *agent.Agent

		expectRes *agent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("9b608bde-53df-11ec-9437-ab8a0e581104"),
			&commonaddress.Address{},
			uuid.FromStringOrNil("54f65714-53df-11ec-9327-470dfe854f0d"),
			uuid.FromStringOrNil("f5b217cc-8c21-11ec-9571-c7a1180c3fdb"),

			&agent.Agent{
				ID:         uuid.FromStringOrNil("9b608bde-53df-11ec-9437-ab8a0e581104"),
				CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				Username:   "test1",
				Name:       "test1 name",
				Detail:     "test1 detail",
				Status:     agent.StatusAvailable,
				Permission: agent.PermissionNone,
				TagIDs:     []uuid.UUID{},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
					},
				},
			},

			&agent.Agent{
				ID:         uuid.FromStringOrNil("89a42670-4c4c-11ec-86ed-9b96390f7668"),
				CustomerID: uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc"),
				Username:   "test1",
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

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &agentHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentGet(gomock.Any(), tt.id).Return(tt.agent, nil)
			mockDB.EXPECT().AgentSetStatus(gomock.Any(), tt.id, agent.StatusRinging).Return(nil)

			for i := 0; i < len(tt.agent.Addresses); i++ {
				mockDB.EXPECT().AgentCallCreate(gomock.Any(), gomock.Any()).Return(nil)
			}

			mockDB.EXPECT().AgentDialCreate(gomock.Any(), gomock.Any()).Return(nil)
			for _, addr := range tt.agent.Addresses {
				callID := uuid.Must(uuid.NewV4())
				mockReq.EXPECT().CallV1CallCreateWithID(gomock.Any(), gomock.Any(), tt.agent.CustomerID, tt.flowID, uuid.Nil, tt.masterCallID, tt.source, &addr).Return(&call.Call{ID: callID}, nil)
			}

			_, err := h.AgentDial(ctx, tt.id, tt.source, tt.flowID, tt.masterCallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
