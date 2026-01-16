package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentAgentGets(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseAgents []amagent.Agent
		expectFilters  map[amagent.Field]any
		expectRes      []*amagent.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			size:  100,
			token: "2021-03-01 01:00:00.995000",

			responseAgents: []amagent.Agent{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("da5b0668-3f9e-11ef-a80b-53bd0e96334d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("dacd54b6-3f9e-11ef-b2fe-f3413488aa55"),
					},
				},
			},
			expectFilters: map[amagent.Field]any{
				amagent.FieldCustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				amagent.FieldDeleted:    false,
			},
			expectRes: []*amagent.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("da5b0668-3f9e-11ef-a80b-53bd0e96334d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("dacd54b6-3f9e-11ef-b2fe-f3413488aa55"),
					},
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseAgents, nil)

			res, err := h.ServiceAgentAgentGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentAgentGet(t *testing.T) {

	type test struct {
		name string

		agent   *amagent.Agent
		agentID uuid.UUID

		responseAgent *amagent.Agent
		expectRes     *amagent.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			agentID: uuid.FromStringOrNil("daf9bea2-3f9e-11ef-9fa2-3bf0ad5fa0ae"),

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("daf9bea2-3f9e-11ef-9fa2-3bf0ad5fa0ae"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			expectRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("daf9bea2-3f9e-11ef-9fa2-3bf0ad5fa0ae"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agentID).Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentAgentGet(ctx, tt.agent, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
