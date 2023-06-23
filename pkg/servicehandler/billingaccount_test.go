package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_billingAccountGet(t *testing.T) {

	tests := []struct {
		name string

		customer         *cscustomer.Customer
		billingAccountID uuid.UUID

		responseBillingAccount *bmaccount.Account
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			},
			uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
			&bmaccount.Account{
				ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
				CustomerID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				TMDelete:   defaultTimestamp,
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

			mockReq.EXPECT().BillingV1AccountGet(ctx, tt.billingAccountID).Return(tt.responseBillingAccount, nil)

			res, err := h.billingAccountGet(ctx, tt.customer, tt.billingAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseBillingAccount) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseBillingAccount, res)
			}
		})
	}
}

func Test_BillingAccountDelete(t *testing.T) {

	tests := []struct {
		name string

		customer         *cscustomer.Customer
		billingAccountID uuid.UUID

		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.WebhookMessage
	}{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("f7269f7a-105e-11ee-87aa-ab1cc9b7e913"),
			},
			billingAccountID: uuid.FromStringOrNil("f7685000-105e-11ee-a7b9-fb7f3da1cef4"),

			responseBillingAccount: &bmaccount.Account{
				ID:         uuid.FromStringOrNil("f7685000-105e-11ee-a7b9-fb7f3da1cef4"),
				CustomerID: uuid.FromStringOrNil("f7269f7a-105e-11ee-87aa-ab1cc9b7e913"),
				TMDelete:   defaultTimestamp,
			},
			expectRes: &bmaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("f7685000-105e-11ee-a7b9-fb7f3da1cef4"),
				CustomerID: uuid.FromStringOrNil("f7269f7a-105e-11ee-87aa-ab1cc9b7e913"),
				TMDelete:   defaultTimestamp,
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

			mockReq.EXPECT().BillingV1AccountGet(ctx, tt.billingAccountID).Return(tt.responseBillingAccount, nil)
			mockReq.EXPECT().BillingV1AccountDelete(ctx, tt.billingAccountID).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountDelete(ctx, tt.customer, tt.billingAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountGets(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string

		responseBillingAcounts []bmaccount.Account
		expectRes              []*bmaccount.WebhookMessage
	}{
		{
			name: "normal",
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("149aaf9e-105d-11ee-aeae-4b0e41ee0cf6"),
			},
			size:  10,
			token: "2020-09-20 03:23:20.995000",

			responseBillingAcounts: []bmaccount.Account{
				{
					ID: uuid.FromStringOrNil("3b598286-105d-11ee-a8e0-d3fe1d127d17"),
				},
				{
					ID: uuid.FromStringOrNil("3b7fc41e-105d-11ee-9b29-a77a519ca3b9"),
				},
			},
			expectRes: []*bmaccount.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("3b598286-105d-11ee-a8e0-d3fe1d127d17"),
				},
				{
					ID: uuid.FromStringOrNil("3b7fc41e-105d-11ee-9b29-a77a519ca3b9"),
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

			mockReq.EXPECT().BillingV1AccountGets(ctx, tt.customer.ID, tt.token, tt.size).Return(tt.responseBillingAcounts, nil)

			res, err := h.BillingAccountGets(ctx, tt.customer, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountCreate(t *testing.T) {

	tests := []struct {
		name string

		customer           *cscustomer.Customer
		billingAccountName string
		detail             string

		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.WebhookMessage
	}{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("64d9d8d8-1060-11ee-9c69-b3b4496d13c7"),
			},
			billingAccountName: "test name",
			detail:             "test detail",

			responseBillingAccount: &bmaccount.Account{
				ID: uuid.FromStringOrNil("650daee2-1060-11ee-aac3-a3c291ad39f5"),
			},
			expectRes: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("650daee2-1060-11ee-aac3-a3c291ad39f5"),
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

			mockReq.EXPECT().BillingV1AccountCreate(ctx, tt.customer.ID, tt.billingAccountName, tt.detail).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountCreate(ctx, tt.customer, tt.billingAccountName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountAddBalanceForce(t *testing.T) {

	tests := []struct {
		name string

		customer           *cscustomer.Customer
		billingAccountName string
		detail             string

		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.WebhookMessage
	}{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: cspermission.PermissionAdmin.ID,
			},
			billingAccountName: "test name",
			detail:             "test detail",

			responseBillingAccount: &bmaccount.Account{
				ID: uuid.FromStringOrNil("650daee2-1060-11ee-aac3-a3c291ad39f5"),
			},
			expectRes: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("650daee2-1060-11ee-aac3-a3c291ad39f5"),
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

			mockReq.EXPECT().BillingV1AccountCreate(ctx, tt.customer.ID, tt.billingAccountName, tt.detail).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountCreate(ctx, tt.customer, tt.billingAccountName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
