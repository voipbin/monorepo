package agenthandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/notifyhandler"
)

func TestAgentGets(t *testing.T) {
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

	tests := []struct {
		name string

		userID uint64
		size   uint64
		token  string
		result []*agent.Agent
	}{
		{
			"normal1",
			1,
			10,
			"2021-11-23 17:55:39.712000",
			[]*agent.Agent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().AgentGets(gomock.Any(), tt.userID, tt.size, tt.token).Return(tt.result, nil)
			_, err := h.AgentGets(ctx, tt.userID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestAgentGetsByTags(t *testing.T) {
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

	tests := []struct {
		name string

		userID uint64
		tags   []uuid.UUID

		result    []*agent.Agent
		expectRes []*agent.Agent
	}{
		{
			"normal",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
			},

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("c47a762e-4c87-11ec-b1d8-531dbb4ebcd2"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
					},
				},
			},
			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("c47a762e-4c87-11ec-b1d8-531dbb4ebcd2"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
					},
				},
			},
		},
		{
			"has 2 agents, 1 selected",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
			},

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("5c61f98a-4c88-11ec-9181-43fb8e090ace"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("5c7cf794-4c88-11ec-a55d-b3af0e75c8e1"),
					UserID: 1,
					TagIDs: []uuid.UUID{},
				},
			},
			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("5c61f98a-4c88-11ec-9181-43fb8e090ace"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
					},
				},
			},
		},
		{
			"has 2 agents, all selected",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
			},

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("7f1d18e2-4c88-11ec-9f6b-4fad140d455c"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("7f3bf4ba-4c88-11ec-ab26-675037d57999"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
			},
			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("7f1d18e2-4c88-11ec-9f6b-4fad140d455c"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("7f3bf4ba-4c88-11ec-ab26-675037d57999"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
			},
		},
		{
			"has 2 agents, none selected",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("9f7746e4-4c88-11ec-9c3a-6b0e38bbc60f"),
			},

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("9f9c03b2-4c88-11ec-ac69-7b00edc54e08"),
					UserID: 1,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("9ffe117e-4c88-11ec-9188-4b98b647fe1d"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("9fd03d44-4c88-11ec-9ebe-3fc386a2a1e6"),
					UserID: 1,
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
			ctx := context.Background()

			mockDB.EXPECT().AgentGets(gomock.Any(), tt.userID, uint64(maxAgentCount), gomock.Any()).Return(tt.result, nil)
			res, err := h.AgentGetsByTagIDs(ctx, tt.userID, tt.tags)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wront match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAgentGetsByTagIDsAndStatus(t *testing.T) {
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

	tests := []struct {
		name string

		userID uint64
		tags   []uuid.UUID
		status agent.Status

		result    []*agent.Agent
		expectRes []*agent.Agent
	}{
		{
			"normal",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("c47a762e-4c87-11ec-b1d8-531dbb4ebcd2"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
					},
				},
			},
			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("c47a762e-4c87-11ec-b1d8-531dbb4ebcd2"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a38c68be-4c87-11ec-a77b-6b95e79bc1bb"),
					},
				},
			},
		},
		{
			"has 2 agents, 1 selected",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("5c61f98a-4c88-11ec-9181-43fb8e090ace"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("5c7cf794-4c88-11ec-a55d-b3af0e75c8e1"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{},
				},
			},
			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("5c61f98a-4c88-11ec-9181-43fb8e090ace"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("5c395822-4c88-11ec-875e-af39deb0b571"),
					},
				},
			},
		},
		{
			"has 2 agents, all selected",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("7f1d18e2-4c88-11ec-9f6b-4fad140d455c"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("7f3bf4ba-4c88-11ec-ab26-675037d57999"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
			},
			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("7f1d18e2-4c88-11ec-9f6b-4fad140d455c"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("7f3bf4ba-4c88-11ec-ab26-675037d57999"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("7f00464a-4c88-11ec-8362-1f73a20620db"),
					},
				},
			},
		},
		{
			"has 2 agents, none selected",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("9f7746e4-4c88-11ec-9c3a-6b0e38bbc60f"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("9f9c03b2-4c88-11ec-ac69-7b00edc54e08"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("9ffe117e-4c88-11ec-9188-4b98b647fe1d"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("9fd03d44-4c88-11ec-9ebe-3fc386a2a1e6"),
					UserID: 1,
					Status: agent.StatusAvailable,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("a02c0a48-4c88-11ec-99da-bb9592c80bf8"),
					},
				},
			},
			[]*agent.Agent{},
		},
		{
			"has 2 agents, none selected by wrong status",
			1,
			[]uuid.UUID{
				uuid.FromStringOrNil("9f7746e4-4c88-11ec-9c3a-6b0e38bbc60f"),
			},
			agent.StatusAvailable,

			[]*agent.Agent{
				{
					ID:     uuid.FromStringOrNil("9f9c03b2-4c88-11ec-ac69-7b00edc54e08"),
					UserID: 1,
					Status: agent.StatusOffline,
					TagIDs: []uuid.UUID{
						uuid.FromStringOrNil("9ffe117e-4c88-11ec-9188-4b98b647fe1d"),
					},
				},
				{
					ID:     uuid.FromStringOrNil("9fd03d44-4c88-11ec-9ebe-3fc386a2a1e6"),
					UserID: 1,
					Status: agent.StatusAway,
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
			ctx := context.Background()

			mockDB.EXPECT().AgentGets(gomock.Any(), tt.userID, uint64(maxAgentCount), gomock.Any()).Return(tt.result, nil)
			res, err := h.AgentGetsByTagIDsAndStatus(ctx, tt.userID, tt.tags, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wront match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestAgentCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &agentHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	tests := []struct {
		name string

		userID     uint64
		username   string
		password   string
		agentName  string
		detail     string
		ringMethod agent.RingMethod
		permission agent.Permission
		tags       []uuid.UUID
		addresses  []cmaddress.Address

		expectRes *agent.Agent
	}{
		{
			"normal",

			1,
			"test1",
			"test1password",
			"test1 name",
			"test1 detail",
			agent.RingMethodRingAll,
			agent.PermissionNone,
			[]uuid.UUID{},
			[]cmaddress.Address{},

			&agent.Agent{
				ID:         uuid.FromStringOrNil("89a42670-4c4c-11ec-86ed-9b96390f7668"),
				UserID:     1,
				Username:   "test1",
				Name:       "test1 name",
				Detail:     "test1 detail",
				Permission: agent.PermissionNone,
				TagIDs:     []uuid.UUID{},
				Addresses:  []cmaddress.Address{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().AgentGetByUsername(gomock.Any(), tt.userID, tt.username).Return(nil, fmt.Errorf("not found"))
			mockDB.EXPECT().AgentCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().AgentGet(gomock.Any(), gomock.Any()).Return(tt.expectRes, nil)

			res, err := h.AgentCreate(ctx, tt.userID, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tags, tt.addresses)
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
