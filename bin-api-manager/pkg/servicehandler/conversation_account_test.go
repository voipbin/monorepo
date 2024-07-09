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
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ConversationAccount(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		responseAccounts []cvaccount.Account
		expectFilters    map[string]string
		expectRes        []*cvaccount.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]cvaccount.Account{
				{
					ID: uuid.FromStringOrNil("e7c38e4c-0048-11ee-b366-4bc7a645f6fb"),
				},
				{
					ID: uuid.FromStringOrNil("e7ec5106-0048-11ee-af79-4b073b23214a"),
				},
			},
			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			[]*cvaccount.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("e7c38e4c-0048-11ee-b366-4bc7a645f6fb"),
				},
				{
					ID: uuid.FromStringOrNil("e7ec5106-0048-11ee-af79-4b073b23214a"),
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

			mockReq.EXPECT().ConversationV1AccountGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseAccounts, nil)
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
				ID:         uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
				ID:         uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),
				CustomerID: uuid.FromStringOrNil("2ace8e8a-0049-11ee-b51e-2b070dbfafef"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),
				CustomerID: uuid.FromStringOrNil("2ace8e8a-0049-11ee-b51e-2b070dbfafef"),
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

		agent       *amagent.Agent
		accountID   uuid.UUID
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
			accountID:   uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
			accountName: "test name",
			detail:      "test detail",
			secret:      "test secret",
			token:       "test token",

			response: &cvaccount.Account{
				ID:         uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
			mockReq.EXPECT().ConversationV1AccountUpdate(ctx, tt.accountID, tt.accountName, tt.detail, tt.secret, tt.token).Return(tt.response, nil)
			res, err := h.ConversationAccountUpdate(ctx, tt.agent, tt.accountID, tt.accountName, tt.detail, tt.secret, tt.token)
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
				ID:         uuid.FromStringOrNil("19beb18c-004a-11ee-a0fb-6325445ef551"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("19beb18c-004a-11ee-a0fb-6325445ef551"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
