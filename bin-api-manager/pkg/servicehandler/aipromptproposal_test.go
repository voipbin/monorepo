package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amaipromptproposal "monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AIPromptProposalCreate(t *testing.T) {
	tests := []struct {
		name string

		agent    *auth.AuthIdentity
		aiID     uuid.UUID
		auditIDs []uuid.UUID
		language string

		responseAI       *amai.AI
		responseProposal *amaipromptproposal.AIPromptProposal

		expectRes *amaipromptproposal.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-aaaa-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("11111111-aaaa-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiID:     uuid.FromStringOrNil("11111111-aaaa-0000-0000-000000000003"),
			auditIDs: []uuid.UUID{uuid.FromStringOrNil("11111111-aaaa-0000-0000-000000000010")},
			language: "en-US",

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("11111111-aaaa-0000-0000-000000000002"),
				},
			},
			responseProposal: &amaipromptproposal.AIPromptProposal{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-aaaa-0000-0000-000000000004"),
				},
			},

			expectRes: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11111111-aaaa-0000-0000-000000000004"),
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIGet(ctx, tt.aiID).Return(tt.responseAI, nil)
			mockReq.EXPECT().AIV1AIPromptProposalCreate(ctx, tt.responseAI.CustomerID, tt.aiID, tt.auditIDs, tt.language).Return(tt.responseProposal, nil)

			res, err := h.AIPromptProposalCreate(ctx, tt.agent, tt.aiID, tt.auditIDs, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIPromptProposalGetsByCustomerID(t *testing.T) {
	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		size   uint64
		token  string
		aiID   uuid.UUID
		status amaipromptproposal.Status

		responseProposals []*amaipromptproposal.AIPromptProposal

		expectFilters map[amaipromptproposal.Field]any
		expectRes     []*amaipromptproposal.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("22222222-aaaa-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("22222222-aaaa-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:  10,
			token: "2020-09-20T03:23:20.995000Z",
			aiID:  uuid.Nil,

			responseProposals: []*amaipromptproposal.AIPromptProposal{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("22222222-aaaa-0000-0000-000000000003"),
					},
				},
			},

			expectFilters: map[amaipromptproposal.Field]any{
				amaipromptproposal.FieldDeleted:    false,
				amaipromptproposal.FieldCustomerID: uuid.FromStringOrNil("22222222-aaaa-0000-0000-000000000002"),
			},
			expectRes: []*amaipromptproposal.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("22222222-aaaa-0000-0000-000000000003"),
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIPromptProposalList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseProposals, nil)

			res, err := h.AIPromptProposalGetsByCustomerID(ctx, tt.agent, tt.size, tt.token, tt.aiID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIPromptProposalGet(t *testing.T) {
	tests := []struct {
		name string

		agent      *auth.AuthIdentity
		proposalID uuid.UUID

		responseProposal *amaipromptproposal.AIPromptProposal

		expectRes *amaipromptproposal.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("33333333-aaaa-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("33333333-aaaa-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			proposalID: uuid.FromStringOrNil("33333333-aaaa-0000-0000-000000000003"),

			responseProposal: &amaipromptproposal.AIPromptProposal{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("33333333-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("33333333-aaaa-0000-0000-000000000002"),
				},
			},

			expectRes: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("33333333-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("33333333-aaaa-0000-0000-000000000002"),
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

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIPromptProposalGet(ctx, tt.proposalID).Return(tt.responseProposal, nil)

			res, err := h.AIPromptProposalGet(ctx, tt.agent, tt.proposalID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIPromptProposalAccept(t *testing.T) {
	tests := []struct {
		name string

		agent      *auth.AuthIdentity
		proposalID uuid.UUID

		responseProposal *amaipromptproposal.AIPromptProposal
		responseAccept   *amaipromptproposal.AIPromptProposal

		expectRes *amaipromptproposal.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			proposalID: uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000003"),

			responseProposal: &amaipromptproposal.AIPromptProposal{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000002"),
				},
				Status: amaipromptproposal.StatusCompleted,
			},
			responseAccept: &amaipromptproposal.AIPromptProposal{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000002"),
				},
				Status: amaipromptproposal.StatusAccepted,
			},

			expectRes: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("44444444-aaaa-0000-0000-000000000002"),
				},
				Status: amaipromptproposal.StatusAccepted,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIPromptProposalGet(ctx, tt.proposalID).Return(tt.responseProposal, nil)
			mockReq.EXPECT().AIV1AIPromptProposalAccept(ctx, tt.responseProposal.CustomerID, tt.proposalID).Return(tt.responseAccept, nil)

			res, err := h.AIPromptProposalAccept(ctx, tt.agent, tt.proposalID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIPromptProposalReject(t *testing.T) {
	tests := []struct {
		name string

		agent      *auth.AuthIdentity
		proposalID uuid.UUID

		responseProposal *amaipromptproposal.AIPromptProposal
		responseReject   *amaipromptproposal.AIPromptProposal

		expectRes *amaipromptproposal.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			proposalID: uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000003"),

			responseProposal: &amaipromptproposal.AIPromptProposal{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000002"),
				},
				Status: amaipromptproposal.StatusCompleted,
			},
			responseReject: &amaipromptproposal.AIPromptProposal{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000002"),
				},
				Status: amaipromptproposal.StatusRejected,
			},

			expectRes: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("55555555-aaaa-0000-0000-000000000002"),
				},
				Status: amaipromptproposal.StatusRejected,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIPromptProposalGet(ctx, tt.proposalID).Return(tt.responseProposal, nil)
			mockReq.EXPECT().AIV1AIPromptProposalReject(ctx, tt.responseProposal.CustomerID, tt.proposalID).Return(tt.responseReject, nil)

			res, err := h.AIPromptProposalReject(ctx, tt.agent, tt.proposalID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIPromptProposalDelete(t *testing.T) {
	tests := []struct {
		name string

		agent      *auth.AuthIdentity
		proposalID uuid.UUID

		responseProposal *amaipromptproposal.AIPromptProposal

		expectRes *amaipromptproposal.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("66666666-aaaa-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("66666666-aaaa-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			proposalID: uuid.FromStringOrNil("66666666-aaaa-0000-0000-000000000003"),

			responseProposal: &amaipromptproposal.AIPromptProposal{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("66666666-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("66666666-aaaa-0000-0000-000000000002"),
				},
			},

			expectRes: &amaipromptproposal.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("66666666-aaaa-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("66666666-aaaa-0000-0000-000000000002"),
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

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIPromptProposalGet(ctx, tt.proposalID).Return(tt.responseProposal, nil)
			mockReq.EXPECT().AIV1AIPromptProposalDelete(ctx, tt.proposalID).Return(tt.responseProposal, nil)

			res, err := h.AIPromptProposalDelete(ctx, tt.agent, tt.proposalID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
