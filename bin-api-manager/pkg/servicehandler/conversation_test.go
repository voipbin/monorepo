package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ConversationListByCustomerID(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	otherAgentID := uuid.FromStringOrNil("a01f2c3a-3001-11f0-9d11-2bd5b4a45af1")

	tests := []struct {
		name      string
		agent     *auth.AuthIdentity
		pageToken string
		pageSize  uint64
		ownerID   uuid.UUID

		responseConversations []cvconversation.Conversation

		expectFilters map[cvconversation.Field]any
		expectRes     []*cvconversation.WebhookMessage
	}{
		{
			name: "admin lists without owner_id filter",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			pageToken: "2020-10-20T01:00:00.995000Z",
			pageSize:  10,
			ownerID:   uuid.Nil,

			responseConversations: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18c13288-ed21-11ec-9d0f-c7be55dc87d7"),
					},
				},
			},
			expectFilters: map[cvconversation.Field]any{
				cvconversation.FieldCustomerID: customerID,
				cvconversation.FieldDeleted:    false,
			},
			expectRes: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18c13288-ed21-11ec-9d0f-c7be55dc87d7"),
					},
				},
			},
		},
		{
			name: "admin lists with owner_id filter (any agent)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			pageToken: "2020-10-20T01:00:00.995000Z",
			pageSize:  10,
			ownerID:   otherAgentID,

			responseConversations: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
					},
				},
			},
			expectFilters: map[cvconversation.Field]any{
				cvconversation.FieldCustomerID: customerID,
				cvconversation.FieldDeleted:    false,
				cvconversation.FieldOwnerID:    otherAgentID,
			},
			expectRes: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
					},
				},
			},
		},
		{
			name: "non-admin agent self-lists with own owner_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			pageToken: "2020-10-20T01:00:00.995000Z",
			pageSize:  10,
			ownerID:   agentID,

			responseConversations: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
					},
				},
			},
			expectFilters: map[cvconversation.Field]any{
				cvconversation.FieldCustomerID: customerID,
				cvconversation.FieldDeleted:    false,
				cvconversation.FieldOwnerID:    agentID,
			},
			expectRes: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
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

			mockReq.EXPECT().ConversationV1ConversationList(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseConversations, nil)
			res, err := h.ConversationGetsByCustomerID(ctx, tt.agent, tt.pageSize, tt.pageToken, tt.ownerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationListByCustomerID_PermissionDenied(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	otherAgentID := uuid.FromStringOrNil("a01f2c3a-3001-11f0-9d11-2bd5b4a45af1")

	tests := []struct {
		name      string
		agent     *auth.AuthIdentity
		pageToken string
		pageSize  uint64
		ownerID   uuid.UUID
	}{
		{
			name: "non-admin agent without owner_id filter",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			pageToken: "2020-10-20T01:00:00.995000Z",
			pageSize:  10,
			ownerID:   uuid.Nil,
		},
		{
			name: "non-admin agent with someone else's owner_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			pageToken: "2020-10-20T01:00:00.995000Z",
			pageSize:  10,
			ownerID:   otherAgentID,
		},
		{
			// Defense-in-depth: a malformed agent identity with a Nil agent ID
			// must NOT pass the self-list gate even if the caller passes
			// owner_id == uuid.Nil. The Nil-check on a.Agent.ID closes a
			// theoretical privilege-escalation path.
			name: "non-admin agent with nil agent id passes nil owner_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.Nil,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			pageToken: "2020-10-20T01:00:00.995000Z",
			pageSize:  10,
			ownerID:   uuid.Nil,
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

			// Permission gate runs before any RPC; ConversationV1ConversationList must NOT be called.
			res, err := h.ConversationGetsByCustomerID(ctx, tt.agent, tt.pageSize, tt.pageToken, tt.ownerID)
			if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
				t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrPermissionDenied, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_ConversationGet(t *testing.T) {

	tests := []struct {
		name           string
		customer       *auth.AuthIdentity
		conversationID uuid.UUID

		response  *cvconversation.Conversation
		expectRes *cvconversation.WebhookMessage
	}{
		{
			"normal",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),

			&cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.response, nil)
			res, err := h.ConversationGet(ctx, tt.customer, tt.conversationID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationUpdate(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	conversationID := uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5")
	owningAgentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		conversationID uuid.UUID
		fileds         map[cvconversation.Field]any

		responseConversation *cvconversation.Conversation
		expectRes            *cvconversation.WebhookMessage
	}{
		{
			name: "admin updates name and detail",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldName:   "test name",
				cvconversation.FieldDetail: "test detail",
			},

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
		},
		{
			name: "admin assigns owner_id (non-nil UUID)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0a4c108-3002-11f0-8a3e-c33b1aef2e49"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: owningAgentID,
			},

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
		},
		{
			name: "admin unassigns owner_id (nil UUID)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0a4c108-3002-11f0-8a3e-c33b1aef2e49"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: uuid.Nil,
			},

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
		},
		{
			name: "manager unassigns owner_id (nil UUID)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d6a401d4-3002-11f0-9c79-b3a64c98c8d9"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerManager,
			}),

			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: uuid.Nil,
			},

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
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

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, tt.conversationID, tt.fileds).Return(tt.responseConversation, nil)
			res, err := h.ConversationUpdate(ctx, tt.agent, tt.conversationID, tt.fileds)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationUpdate_PermissionDenied(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	conversationID := uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5")
	owningAgentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	otherAgentID := uuid.FromStringOrNil("a01f2c3a-3001-11f0-9d11-2bd5b4a45af1")

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		conversationID uuid.UUID
		fileds         map[cvconversation.Field]any

		responseConversation *cvconversation.Conversation
	}{
		{
			name: "owning agent attempts to assign someone else (non-nil owner_id)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: otherAgentID,
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
		},
		{
			name: "owning agent attempts to assign self (non-nil owner_id == self)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: owningAgentID,
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
		},
		{
			name: "owning agent attempts to update name only",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldName: "renamed by agent",
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
		},
		{
			name: "owning agent attempts combined self-unassign and name change",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: uuid.Nil,
				cvconversation.FieldName:    "renamed by agent",
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
		},
		{
			name: "owning agent attempts OpenAPI-bypass payload (owner_id nil + owner_type)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID:   uuid.Nil,
				cvconversation.FieldOwnerType: string(commonidentity.OwnerTypeAgent),
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
		},
		{
			name: "non-owning agent attempts unassign (nil owner_id)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: uuid.Nil,
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
		},
		{
			name: "non-owning agent attempts to update name",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldName: "renamed by other agent",
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
				},
			},
		},
		{
			name: "agent on unassigned conversation attempts unassign",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: uuid.Nil,
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeNone,
					OwnerID:   uuid.Nil,
				},
			},
		},
		{
			name: "owning agent calls PUT — now rejected (breaking change)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID: conversationID,
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: uuid.Nil,
			},
			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   owningAgentID,
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

			// conversationGet is called before the permission gate; only it should be mocked.
			// ConversationV1ConversationUpdate must NOT be called for denied requests — no mock.
			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			res, err := h.ConversationUpdate(ctx, tt.agent, tt.conversationID, tt.fileds)
			if !errors.Is(err, serviceerrors.ErrPermissionDenied) {
				t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrPermissionDenied, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_ConversationUnassign(t *testing.T) {
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
			name: "admin unassigns assigned conversation",
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
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
		},
		{
			name: "manager unassigns assigned conversation",
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
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
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
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
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
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
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
			h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
			ctx := context.Background()

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			if tt.expectCallUpdate {
				mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, tt.conversationID, unassignFields).Return(tt.responseUpdated, nil)
			}

			res, err := h.ConversationUnassign(ctx, tt.agent, tt.conversationID)
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
