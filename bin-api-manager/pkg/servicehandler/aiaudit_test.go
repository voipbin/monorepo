package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amaiaudit "monorepo/bin-ai-manager/models/aiaudit"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AIAuditCreate(t *testing.T) {
	tests := []struct {
		name string

		agent    *auth.AuthIdentity
		aicallID uuid.UUID
		language string

		responseAIcall   *amaicall.AIcall
		responseAIAudits []*amaiaudit.AIAudit

		expectRes []*amaiaudit.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aicallID: uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-000000000003"),
			language: "en-US",

			responseAIcall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-000000000002"),
				},
			},
			responseAIAudits: []*amaiaudit.AIAudit{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-000000000004"),
					},
				},
			},

			expectRes: []*amaiaudit.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b3e2a1c0-0000-0000-0000-000000000004"),
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

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAIcall, nil)
			mockReq.EXPECT().AIV1AIAuditCreate(ctx, tt.responseAIcall.CustomerID, tt.aicallID, tt.language).Return(tt.responseAIAudits, nil)

			res, err := h.AIAuditCreate(ctx, tt.agent, tt.aicallID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIAuditGetsByCustomerID(t *testing.T) {
	tests := []struct {
		name string

		agent    *auth.AuthIdentity
		size     uint64
		token    string
		aicallID uuid.UUID
		aiID     uuid.UUID

		mockToken        string
		responseAIAudits []*amaiaudit.AIAudit

		expectFilters map[amaiaudit.Field]any
		expectRes     []*amaiaudit.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c4f3b2a1-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c4f3b2a1-0000-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:     10,
			token:    "2020-09-20T03:23:20.995000Z",
			aicallID: uuid.Nil,
			aiID:     uuid.Nil,

			responseAIAudits: []*amaiaudit.AIAudit{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c4f3b2a1-0000-0000-0000-000000000003"),
					},
				},
			},

			expectFilters: map[amaiaudit.Field]any{
				amaiaudit.FieldDeleted:    false,
				amaiaudit.FieldCustomerID: uuid.FromStringOrNil("c4f3b2a1-0000-0000-0000-000000000002"),
			},
			expectRes: []*amaiaudit.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c4f3b2a1-0000-0000-0000-000000000003"),
					},
				},
			},
		},
		{
			name: "empty_token_uses_current_time",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c4f3b2a1-0000-0000-0000-000000000011"),
					CustomerID: uuid.FromStringOrNil("c4f3b2a1-0000-0000-0000-000000000012"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:     10,
			token:    "",
			aicallID: uuid.Nil,
			aiID:     uuid.Nil,

			mockToken: "2020-09-20T03:23:20.995000Z",

			responseAIAudits: []*amaiaudit.AIAudit{},

			expectFilters: map[amaiaudit.Field]any{
				amaiaudit.FieldDeleted:    false,
				amaiaudit.FieldCustomerID: uuid.FromStringOrNil("c4f3b2a1-0000-0000-0000-000000000012"),
			},
			expectRes: []*amaiaudit.WebhookMessage{},
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

			effectiveToken := tt.token
			if tt.token == "" {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.mockToken)
				effectiveToken = tt.mockToken
			}

			mockReq.EXPECT().AIV1AIAuditList(ctx, effectiveToken, tt.size, tt.expectFilters).Return(tt.responseAIAudits, nil)

			res, err := h.AIAuditGetsByCustomerID(ctx, tt.agent, tt.size, tt.token, tt.aicallID, tt.aiID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIAuditGet(t *testing.T) {
	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		aiauditID uuid.UUID

		responseAIAudit *amaiaudit.AIAudit
		responseErr     error

		expectRes *amaiaudit.WebhookMessage
		expectErr bool
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiauditID: uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000003"),

			responseAIAudit: &amaiaudit.AIAudit{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000002"),
				},
			},

			expectRes: &amaiaudit.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name: "error_not_found",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000011"),
					CustomerID: uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000012"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiauditID: uuid.FromStringOrNil("d5a4c3b2-0000-0000-0000-000000000013"),

			responseAIAudit: nil,
			responseErr:     fmt.Errorf("not found"),

			expectErr: true,
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

			mockReq.EXPECT().AIV1AIAuditGet(ctx, tt.aiauditID).Return(tt.responseAIAudit, tt.responseErr)

			res, err := h.AIAuditGet(ctx, tt.agent, tt.aiauditID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIAuditDelete(t *testing.T) {
	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		aiauditID uuid.UUID

		responseAIAudit *amaiaudit.AIAudit
		responseGetErr  error
		responseDelErr  error

		expectRes *amaiaudit.WebhookMessage
		expectErr bool
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000002"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiauditID: uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000003"),

			responseAIAudit: &amaiaudit.AIAudit{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000002"),
				},
			},

			expectRes: &amaiaudit.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000003"),
					CustomerID: uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000002"),
				},
			},
		},
		{
			name: "error_get_not_found",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000011"),
					CustomerID: uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000012"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aiauditID: uuid.FromStringOrNil("e6b5d4c3-0000-0000-0000-000000000013"),

			responseAIAudit: nil,
			responseGetErr:  fmt.Errorf("not found"),

			expectErr: true,
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

			mockReq.EXPECT().AIV1AIAuditGet(ctx, tt.aiauditID).Return(tt.responseAIAudit, tt.responseGetErr)

			if tt.responseGetErr == nil {
				mockReq.EXPECT().AIV1AIAuditDelete(ctx, tt.aiauditID).Return(tt.responseAIAudit, tt.responseDelErr)
			}

			res, err := h.AIAuditDelete(ctx, tt.agent, tt.aiauditID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
