package accounthandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID

		responseGet      *account.Account
		responseCustomer *cscustomer.Customer

		expectRes *account.Account
	}{
		{
			"has a cache",
			uuid.FromStringOrNil("3b24255a-e60b-11ec-9815-5f679b51ac4d"),

			&account.Account{
				ID:         uuid.FromStringOrNil("3b24255a-e60b-11ec-9815-5f679b51ac4d"),
				LineSecret: "36d4fc54-e60b-11ec-bd30-ef0549a549a1",
				LineToken:  "371d4e64-e60b-11ec-aea1-f3e59c17f7c3",
			},
			nil,

			&account.Account{
				ID:         uuid.FromStringOrNil("3b24255a-e60b-11ec-9815-5f679b51ac4d"),
				LineSecret: "36d4fc54-e60b-11ec-bd30-ef0549a549a1",
				LineToken:  "371d4e64-e60b-11ec-aea1-f3e59c17f7c3",
			},
		},
		{
			"has no cache",
			uuid.FromStringOrNil("3b24255a-e60b-11ec-9815-5f679b51ac4d"),

			nil,
			&cscustomer.Customer{
				ID:         uuid.FromStringOrNil("dd20b7a2-ed3a-11ec-af5c-d70968df34d7"),
				LineSecret: "36d4fc54-e60b-11ec-bd30-ef0549a549a1",
				LineToken:  "371d4e64-e60b-11ec-aea1-f3e59c17f7c3",
			},

			&account.Account{
				ID:         uuid.FromStringOrNil("dd20b7a2-ed3a-11ec-af5c-d70968df34d7"),
				LineSecret: "36d4fc54-e60b-11ec-bd30-ef0549a549a1",
				LineToken:  "371d4e64-e60b-11ec-aea1-f3e59c17f7c3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			if tt.responseGet != nil {
				mockDB.EXPECT().AccountGet(ctx, tt.customerID).Return(tt.responseGet, nil)
			} else {
				mockDB.EXPECT().AccountGet(ctx, tt.customerID).Return(nil, fmt.Errorf(""))
				mockReq.EXPECT().CSV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
				mockDB.EXPECT().AccountSet(ctx, gomock.Any()).Return(nil)
			}

			res, err := h.Get(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func Test_Set(t *testing.T) {

	tests := []struct {
		name string

		account *account.Account
	}{
		{
			"normal",

			&account.Account{
				ID:         uuid.FromStringOrNil("8aed14fc-e60b-11ec-9afc-731273d4009f"),
				LineSecret: "8b2b9484-e60b-11ec-a88e-eb65f2edfe74",
				LineToken:  "8b595d9c-e60b-11ec-9529-63aac373fb91",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockDB.EXPECT().AccountSet(gomock.Any(), tt.account).Return(nil)

			if err := h.Set(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateByCustomer(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer

		expectReqAccount *account.Account
		responseAccount  *account.Account
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("51a5325a-833d-11ec-8759-c79414e8a44e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},

			&account.Account{
				ID: uuid.FromStringOrNil("51a5325a-833d-11ec-8759-c79414e8a44e"),
			},
			&account.Account{
				ID: uuid.FromStringOrNil("51a5325a-833d-11ec-8759-c79414e8a44e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockDB.EXPECT().AccountSet(gomock.Any(), tt.expectReqAccount).Return(nil)
			mockDB.EXPECT().AccountGet(gomock.Any(), tt.customer.ID).Return(tt.responseAccount, nil)

			res, err := h.UpdateByCustomer(ctx, tt.customer)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseAccount, res)
			}
		})
	}
}
