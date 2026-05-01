package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentConversationGet(t *testing.T) {

	type test struct {
		name string

		agent          *auth.AuthIdentity
		conversationID uuid.UUID

		responseConversation *cvconversation.Conversation
		expectErr            error
		expectRes            *cvconversation.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			conversationID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
		},
		{
			name: "admin gets conversation owned by another agent",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaaaaaa-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			conversationID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
		},
		{
			name: "manager gets conversation owned by another agent",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bbbbbbbb-3b9f-11ef-98ac-db226570f09b"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			conversationID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
		},
		{
			name: "regular agent denied for conversation it does not own",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			conversationID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("aaaaaaaa-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			expectErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "admin denied for conversation in different customer",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaaaaaa-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			conversationID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
					CustomerID: uuid.FromStringOrNil("cccccccc-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("dddddddd-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			expectErr: serviceerrors.ErrPermissionDenied,
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

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)

			res, err := h.ServiceAgentConversationGet(ctx, tt.agent, tt.conversationID)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentConversationList(t *testing.T) {

	type test struct {
		name string

		agent *auth.AuthIdentity
		size  uint64
		token string

		responseConversations []cvconversation.Conversation

		expectFilters map[cvconversation.Field]any
		expectErr     error
		expectRes     []*cvconversation.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			size:  100,
			token: "2021-03-01T01:00:00.995000Z",

			responseConversations: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("620bce9e-3ed2-11ef-b45a-3f6e2898153d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("62a1ec8a-3ed2-11ef-bb8c-a788ea1ad2ad"),
					},
				},
			},

			expectFilters: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				cvconversation.FieldDeleted: false,
			},
			expectRes: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("620bce9e-3ed2-11ef-b45a-3f6e2898153d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("62a1ec8a-3ed2-11ef-bb8c-a788ea1ad2ad"),
					},
				},
			},
		},
		{
			name: "admin sees all conversations (no owner filter)",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:  100,
			token: "2021-03-01T01:00:00.995000Z",

			responseConversations: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("620bce9e-3ed2-11ef-b45a-3f6e2898153d"),
					},
				},
			},

			expectFilters: map[cvconversation.Field]any{
				cvconversation.FieldCustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				cvconversation.FieldDeleted:    false,
			},
			expectRes: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("620bce9e-3ed2-11ef-b45a-3f6e2898153d"),
					},
				},
			},
		},
		{
			name: "manager sees all conversations (no owner filter)",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09b"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			size:  100,
			token: "2021-03-01T01:00:00.995000Z",

			responseConversations: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("62a1ec8a-3ed2-11ef-bb8c-a788ea1ad2ad"),
					},
				},
			},

			expectFilters: map[cvconversation.Field]any{
				cvconversation.FieldCustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				cvconversation.FieldDeleted:    false,
			},
			expectRes: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("62a1ec8a-3ed2-11ef-bb8c-a788ea1ad2ad"),
					},
				},
			},
		},
		{
			name:      "non-agent caller is rejected",
			agent:     &auth.AuthIdentity{Type: auth.TypeAccesskey},
			size:      10,
			token:     "2021-03-01T01:00:00.995000Z",
			expectErr: serviceerrors.ErrAuthenticationRequired,
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

			if tt.expectErr == nil {
				mockReq.EXPECT().ConversationV1ConversationList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseConversations, nil)
			}

			res, err := h.ServiceAgentConversationList(ctx, tt.agent, tt.size, tt.token)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentConversationUpdate(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	conversationID := uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")

	conversation := &cvconversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         conversationID,
			CustomerID: customerID,
		},
	}
	updateFields := map[cvconversation.Field]any{
		cvconversation.FieldName: "updated name",
	}

	type test struct {
		name                 string
		agent                *auth.AuthIdentity
		conversationID       uuid.UUID
		fields               map[cvconversation.Field]any
		responseConversation *cvconversation.Conversation
		expectCallUpdate     bool
		responseUpdated      *cvconversation.Conversation
		expectErr            error
		expectRes            *cvconversation.WebhookMessage
	}

	tests := []test{
		{
			name: "admin updates conversation",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			conversationID:       conversationID,
			fields:               updateFields,
			responseConversation: conversation,
			expectCallUpdate:     true,
			responseUpdated:      conversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
		},
		{
			name: "manager updates conversation",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			conversationID:       conversationID,
			fields:               updateFields,
			responseConversation: conversation,
			expectCallUpdate:     true,
			responseUpdated:      conversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
		},
		{
			name: "agent (non-admin) — permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID:       conversationID,
			fields:               updateFields,
			responseConversation: conversation,
			expectCallUpdate:     false,
			expectErr:            serviceerrors.ErrPermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
			ctx := context.Background()

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			if tt.expectCallUpdate {
				mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, tt.conversationID, tt.fields).Return(tt.responseUpdated, nil)
			}

			res, err := h.ServiceAgentConversationUpdate(ctx, tt.agent, tt.conversationID, tt.fields)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong result.\nexpect: %v\ngot:    %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentConversationUnassign(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	conversationID := uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5")
	owningAgentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	otherAgentID := uuid.FromStringOrNil("a01f2c3a-3001-11f0-9d11-2bd5b4a45af1")

	assignedConversation := &cvconversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         conversationID,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   owningAgentID,
		},
	}
	unassignedConversation := &cvconversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         conversationID,
			CustomerID: customerID,
		},
	}
	unassignFields := map[cvconversation.Field]any{
		cvconversation.FieldOwnerID: uuid.Nil,
	}

	type test struct {
		name                 string
		agent                *auth.AuthIdentity
		conversationID       uuid.UUID
		responseConversation *cvconversation.Conversation
		expectCallUpdate     bool
		responseUpdated      *cvconversation.Conversation
		expectErr            error
		expectRes            *cvconversation.WebhookMessage
	}

	tests := []test{
		{
			name: "admin unassigns",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0a4c108-3002-11f0-8a3e-c33b1aef2e49"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			conversationID:       conversationID,
			responseConversation: assignedConversation,
			expectCallUpdate:     true,
			responseUpdated:      unassignedConversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
			},
		},
		{
			name: "manager unassigns",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d6a401d4-3002-11f0-9c79-b3a64c98c8d9"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			conversationID:       conversationID,
			responseConversation: assignedConversation,
			expectCallUpdate:     true,
			responseUpdated:      unassignedConversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
			},
		},
		{
			name: "owning agent self-unassigns",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID:       conversationID,
			responseConversation: assignedConversation,
			expectCallUpdate:     true,
			responseUpdated:      unassignedConversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
			},
		},
		{
			name: "non-owning agent — permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID:       conversationID,
			responseConversation: assignedConversation,
			expectCallUpdate:     false,
			expectErr:            serviceerrors.ErrPermissionDenied,
		},
		{
			name: "owning agent on already-unassigned conversation — permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID:       conversationID,
			responseConversation: unassignedConversation,
			expectCallUpdate:     false,
			expectErr:            serviceerrors.ErrPermissionDenied,
		},
		{
			name: "admin on already-unassigned conversation — idempotent 200",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0a4c108-3002-11f0-8a3e-c33b1aef2e49"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			conversationID:       conversationID,
			responseConversation: unassignedConversation,
			expectCallUpdate:     true,
			responseUpdated:      unassignedConversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
			ctx := context.Background()

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			if tt.expectCallUpdate {
				mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, tt.conversationID, unassignFields).Return(tt.responseUpdated, nil)
			}

			res, err := h.ServiceAgentConversationUnassign(ctx, tt.agent, tt.conversationID)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong result.\nexpect: %v\ngot:    %v", tt.expectRes, res)
			}
		})
	}
}
