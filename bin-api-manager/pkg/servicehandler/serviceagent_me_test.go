package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ServiceAgentMeGet(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agent.ID.Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeGet(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdate(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		agentName  string
		detail     string
		ringMethod amagent.RingMethod

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			agentName:  "update name",
			detail:     "update detail",
			ringMethod: amagent.RingMethodRingAll,

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentUpdate(ctx, tt.agent.ID, tt.agentName, tt.detail, tt.ringMethod.Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeUpdate(ctx, tt.agent, tt.agentName, tt.detail, tt.ringMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdateAddresses(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		addresses []commonaddress.Address

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			addresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+123456789",
				},
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentUpdateAddresses(ctx, tt.agent.ID, tt.addresses.Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeUpdateAddresses(ctx, tt.agent, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		status amagent.Status

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			status: amagent.StatusAvailable,

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentUpdateStatus(ctx, tt.agent.ID, tt.status.Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeUpdateStatus(ctx, tt.agent, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdatePassword(t *testing.T) {

	tests := []struct {
		name string

		agent    *amagent.Agent
		password string

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			password: "update_password",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentUpdatePassword(ctx, gomock.Any(), tt.agent.ID, tt.password.Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeUpdatePassword(ctx, tt.agent, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}
