package accounthandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string
		id   uuid.UUID

		responseGet *account.Account

		expectRes *account.Account
	}{
		{
			"normal",
			uuid.FromStringOrNil("6515992a-833c-11ec-b53e-ff69ef240833"),
			&account.Account{
				ID:            uuid.FromStringOrNil("6515992a-833c-11ec-b53e-ff69ef240833"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},

			&account.Account{
				ID:            uuid.FromStringOrNil("6515992a-833c-11ec-b53e-ff69ef240833"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
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

			mockDB.EXPECT().AccountGet(gomock.Any(), tt.id).Return(tt.responseGet, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func Test_GetErrorDB(t *testing.T) {

	tests := []struct {
		name string
		id   uuid.UUID

		responseGet *cscustomer.Customer
	}{
		{
			"normal",
			uuid.FromStringOrNil("bfe9d244-833c-11ec-943e-67954ab4201c"),
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("bfe9d244-833c-11ec-943e-67954ab4201c"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
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

			mockDB.EXPECT().AccountGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf(""))
			mockReq.EXPECT().CustomerV1CustomerGet(gomock.Any(), tt.id).Return(tt.responseGet, nil)

			tmp := account.CreateAccountFromCustomer(tt.responseGet)
			mockDB.EXPECT().AccountSet(gomock.Any(), tmp).Return(nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tmp) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tmp, res)
			}

		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string
		data *account.Account
	}{
		{
			"normal",
			&account.Account{
				ID:            uuid.FromStringOrNil("6515992a-833c-11ec-b53e-ff69ef240833"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
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

			mockDB.EXPECT().AccountSet(gomock.Any(), tt.data).Return(nil)

			if err := h.Update(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_UpdateByCustomer(t *testing.T) {

	tests := []struct {
		name string

		customerInfo *cscustomer.Customer
		data         *account.Account
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("51a5325a-833d-11ec-8759-c79414e8a44e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
			&account.Account{
				ID:            uuid.FromStringOrNil("51a5325a-833d-11ec-8759-c79414e8a44e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
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

			tmp := account.CreateAccountFromCustomer(tt.customerInfo)
			mockDB.EXPECT().AccountSet(gomock.Any(), tmp).Return(nil)

			res, err := h.UpdateByCustomer(ctx, tt.customerInfo)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.data) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.data, res)
			}

		})
	}
}
