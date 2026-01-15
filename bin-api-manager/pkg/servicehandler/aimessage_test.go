package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	ammessage "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AImessageCreate(t *testing.T) {

	tests := []struct {
		name string

		agent    *amagent.Agent
		aicallID uuid.UUID
		role     ammessage.Role
		content  string

		responseAIcall *amaicall.AIcall
		response       *ammessage.Message
		expectRes      *ammessage.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			aicallID: uuid.FromStringOrNil("556b07aa-f31c-11ef-8b45-8782c358d446"),
			role:     ammessage.RoleUser,
			content:  "test text",

			responseAIcall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("556b07aa-f31c-11ef-8b45-8782c358d446"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			response: &ammessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("55c7ed9e-f31c-11ef-9fcd-9fde09fbb4e8"),
				},
			},
			expectRes: &ammessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("55c7ed9e-f31c-11ef-9fcd-9fde09fbb4e8"),
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

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAIcall, nil)
			mockReq.EXPECT().AIV1MessageSend(ctx, tt.aicallID, tt.role, tt.content, true, false, 30000).Return(tt.response, nil)

			res, err := h.AImessageCreate(ctx, tt.agent, tt.aicallID, tt.role, tt.content)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_AImessageGetsByAIcallID(t *testing.T) {

	tests := []struct {
		name string

		agent    *amagent.Agent
		aicallID uuid.UUID
		size     uint64
		token    string

		responseAIcall *amaicall.AIcall
		response       []ammessage.Message

		expectFilters map[ammessage.Field]any
		expectRes     []*ammessage.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			aicallID: uuid.FromStringOrNil("24d250de-f31d-11ef-846e-9ba3307567d6"),
			size:     10,
			token:    "2020-09-20 03:23:20.995000",

			responseAIcall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("24d250de-f31d-11ef-846e-9ba3307567d6"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			response: []ammessage.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("252bafd0-f31d-11ef-983f-b72b407260c8"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("254f2e6a-f31d-11ef-8721-0389c730e82e"),
					},
				},
			},

			expectFilters: map[ammessage.Field]any{
				ammessage.FieldDeleted:  false,
				ammessage.FieldAIcallID: uuid.FromStringOrNil("24d250de-f31d-11ef-846e-9ba3307567d6"),
			},
			expectRes: []*ammessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("252bafd0-f31d-11ef-983f-b72b407260c8"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("254f2e6a-f31d-11ef-8721-0389c730e82e"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAIcall, nil)
			mockReq.EXPECT().AIV1MessageGetsByAIcallID(ctx, tt.aicallID, tt.token, tt.size, tt.expectFilters).Return(tt.response, nil)

			res, err := h.AImessageGetsByAIcallID(ctx, tt.agent, tt.aicallID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AImessageGet(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		messageID uuid.UUID

		response  *ammessage.Message
		expectRes *ammessage.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			messageID: uuid.FromStringOrNil("b8bf966c-f31d-11ef-ba3b-834c48052c25"),

			response: &ammessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b8bf966c-f31d-11ef-ba3b-834c48052c25"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &ammessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b8bf966c-f31d-11ef-ba3b-834c48052c25"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1MessageGet(ctx, tt.messageID).Return(tt.response, nil)

			res, err := h.AImessageGet(ctx, tt.agent, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AImessageDelete(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		messageID uuid.UUID

		response *ammessage.Message

		expectRes *ammessage.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			messageID: uuid.FromStringOrNil("b8e73d98-f31d-11ef-8b29-8b31b17b57dc"),

			response: &ammessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b8e73d98-f31d-11ef-8b29-8b31b17b57dc"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &ammessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b8e73d98-f31d-11ef-8b29-8b31b17b57dc"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().AIV1MessageGet(ctx, tt.messageID).Return(tt.response, nil)
			mockReq.EXPECT().AIV1MessageDelete(ctx, tt.messageID).Return(tt.response, nil)

			_, err := h.AImessageDelete(ctx, tt.agent, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
