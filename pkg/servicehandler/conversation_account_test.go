package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cvaccount "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ConversationAccountGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []cvaccount.Account
		expectRes []*cvaccount.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("e77e86c6-0048-11ee-b7ae-4fae8756eb1a"),
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

			mockReq.EXPECT().ConversationV1AccountGetsByCustomerID(ctx, tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)
			res, err := h.ConversationAccountGetsByCustomerID(ctx, tt.customer, tt.pageSize, tt.pageToken)
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
		customer  *cscustomer.Customer
		accountID uuid.UUID

		response  *cvaccount.Account
		expectRes *cvaccount.WebhookMessage
	}{
		{
			name: "normal",
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("2ace8e8a-0049-11ee-b51e-2b070dbfafef"),
			},

			accountID: uuid.FromStringOrNil("2aee5ae4-0049-11ee-9204-cb301aa1dca8"),

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

			mockReq.EXPECT().ConversationV1AccountGet(ctx, tt.accountID).Return(tt.response, nil)
			res, err := h.ConversationAccountGet(ctx, tt.customer, tt.accountID)
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
		name     string
		customer *cscustomer.Customer

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
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("7da4121a-0049-11ee-bf60-c754fe324319"),
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

			mockReq.EXPECT().ConversationV1AccountCreate(ctx, tt.customer.ID, tt.accountType, tt.accountName, tt.detail, tt.secret, tt.token).Return(tt.response, nil)
			res, err := h.ConversationAccountCreate(ctx, tt.customer, tt.accountType, tt.accountName, tt.detail, tt.secret, tt.token)
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

		customer    *cscustomer.Customer
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
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("d44ac280-0049-11ee-8c4a-5bdd07a7fdc7"),
			},
			accountID:   uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
			accountName: "test name",
			detail:      "test detail",
			secret:      "test secret",
			token:       "test token",

			response: &cvaccount.Account{
				ID:         uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
				CustomerID: uuid.FromStringOrNil("d44ac280-0049-11ee-8c4a-5bdd07a7fdc7"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("d4217f6a-0049-11ee-bedc-df9b0d890304"),
				CustomerID: uuid.FromStringOrNil("d44ac280-0049-11ee-8c4a-5bdd07a7fdc7"),
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
			res, err := h.ConversationAccountUpdate(ctx, tt.customer, tt.accountID, tt.accountName, tt.detail, tt.secret, tt.token)
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

		customer  *cscustomer.Customer
		accountID uuid.UUID

		response  *cvaccount.Account
		expectRes *cvaccount.WebhookMessage
	}{
		{
			name: "normal",
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("19895a14-004a-11ee-a03e-abcf4d9a4887"),
			},
			accountID: uuid.FromStringOrNil("19beb18c-004a-11ee-a0fb-6325445ef551"),

			response: &cvaccount.Account{
				ID:         uuid.FromStringOrNil("19beb18c-004a-11ee-a0fb-6325445ef551"),
				CustomerID: uuid.FromStringOrNil("19895a14-004a-11ee-a03e-abcf4d9a4887"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("19beb18c-004a-11ee-a0fb-6325445ef551"),
				CustomerID: uuid.FromStringOrNil("19895a14-004a-11ee-a03e-abcf4d9a4887"),
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
			res, err := h.ConversationAccountDelete(ctx, tt.customer, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
