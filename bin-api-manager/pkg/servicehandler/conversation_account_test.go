package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cvaccount "monorepo/bin-conversation-manager/models/account"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ConversationAccountListByCustomerID(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		responseAccounts []cvaccount.Account
		expectFilters    map[cvaccount.Field]any
		expectRes        []*cvaccount.WebhookMessage
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
			pageToken: "2020-10-20T01:00:00.995000Z",
			pageSize:  10,

			responseAccounts: []cvaccount.Account{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e7c38e4c-0048-11ee-b366-4bc7a645f6fb"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e7ec5106-0048-11ee-af79-4b073b23214a"),
					},
				},
			},
			expectFilters: map[cvaccount.Field]any{
				cvaccount.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				cvaccount.FieldDeleted:    false,
			},
			expectRes: []*cvaccount.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e7c38e4c-0048-11ee-b366-4bc7a645f6fb"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e7ec5106-0048-11ee-af79-4b073b23214a"),
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

			mockReq.EXPECT().ConversationV1AccountList(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseAccounts, nil)
			res, err := h.ConversationAccountGetsByCustomerID(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationAccountGet(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		accountID uuid.UUID

		response  *cvaccount.Account
		expectRes *cvaccount.WebhookMessage
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

			accountID: uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),

			response: &cvaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cvaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),
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

			mockReq.EXPECT().ConversationV1AccountGet(ctx, tt.accountID).Return(tt.response, nil)
			res, err := h.ConversationAccountGet(ctx, tt.agent, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationAccountCreate(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		accountType cvaccount.Type
		accountName string
		detail      string
		secret      string
		token       string

		response  *cvaccount.Account
		expectRes *cvaccount.WebhookMessage
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

			accountType: cvaccount.TypeLine,
			accountName: "test name",
			detail:      "test detail",
			secret:      "test secret",
			token:       "test token",

			response: &cvaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),
					CustomerID: uuid.FromStringOrNil("2ace8e8a-0049-11ee-b51e-2b070dbfafef"),
				},
			},
			expectRes: &cvaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),
					CustomerID: uuid.FromStringOrNil("2ace8e8a-0049-11ee-b51e-2b070dbfafef"),
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

			mockReq.EXPECT().ConversationV1AccountCreate(ctx, tt.agent.CustomerID, tt.accountType, tt.accountName, tt.detail, tt.secret, tt.token).Return(tt.response, nil)
			res, err := h.ConversationAccountCreate(ctx, tt.agent, tt.accountType, tt.accountName, tt.detail, tt.secret, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationAccountUpdate(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		accountID uuid.UUID
		fileds    map[cvaccount.Field]any

		response  *cvaccount.Account
		expectRes *cvaccount.WebhookMessage
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
			accountID: uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
			fileds: map[cvaccount.Field]any{
				cvaccount.FieldName:   "test name",
				cvaccount.FieldDetail: "test detail",
				cvaccount.FieldSecret: "test secret",
				cvaccount.FieldToken:  "test token",
			},

			response: &cvaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cvaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
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

			mockReq.EXPECT().ConversationV1AccountGet(ctx, tt.accountID).Return(tt.response, nil)
			mockReq.EXPECT().ConversationV1AccountUpdate(ctx, tt.accountID, tt.fileds).Return(tt.response, nil)
			res, err := h.ConversationAccountUpdate(ctx, tt.agent, tt.accountID, tt.fileds)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationAccountDelete(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		accountID uuid.UUID

		response  *cvaccount.Account
		expectRes *cvaccount.WebhookMessage
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
			accountID: uuid.FromStringOrNil("19beb18c-004a-11ee-a0fb-6325445ef551"),

			response: &cvaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("19beb18c-004a-11ee-a0fb-6325445ef551"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cvaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("19beb18c-004a-11ee-a0fb-6325445ef551"),
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

			mockReq.EXPECT().ConversationV1AccountGet(ctx, tt.accountID).Return(tt.response, nil)
			mockReq.EXPECT().ConversationV1AccountDelete(ctx, tt.accountID).Return(tt.response, nil)
			res, err := h.ConversationAccountDelete(ctx, tt.agent, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
