package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_billingAccountGet(t *testing.T) {

	tests := []struct {
		name string

		agent            *amagent.Agent
		billingAccountID uuid.UUID

		responseBillingAccount *bmaccount.Account
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
			uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
			&bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
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

			res, err := h.billingAccountGet(ctx, tt.billingAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseBillingAccount) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseBillingAccount, res)
			}
		})
	}
}

func Test_billingAccountGet_Deleted(t *testing.T) {

	tests := []struct {
		name string

		billingAccountID uuid.UUID

		responseBillingAccount *bmaccount.Account
	}{
		{
			"deleted billing account",
			uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
			&bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: func() *time.Time { t := time.Now(); return &t }(),
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

			_, err := h.billingAccountGet(ctx, tt.billingAccountID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountGet_NoPermission(t *testing.T) {

	tests := []struct {
		name string

		agent            *amagent.Agent
		billingAccountID uuid.UUID
	}{
		{
			name: "customer admin has no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			billingAccountID: uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
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

			_, err := h.BillingAccountGet(ctx, tt.agent, tt.billingAccountID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		agent              *amagent.Agent
		billingAccountID   uuid.UUID
		billingAccountName string
		detail             string

		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.Account
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			billingAccountID:   uuid.FromStringOrNil("91aea826-4cdc-11ee-9e0f-7bde2e963cc8"),
			billingAccountName: "test name",
			detail:             "test detail",

			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91aea826-4cdc-11ee-9e0f-7bde2e963cc8"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},
			expectRes: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91aea826-4cdc-11ee-9e0f-7bde2e963cc8"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
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

			mockReq.EXPECT().BillingV1AccountUpdateBasicInfo(ctx, tt.billingAccountID, tt.billingAccountName, tt.detail).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountUpdateBasicInfo(ctx, tt.agent, tt.billingAccountID, tt.billingAccountName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountUpdatePaymentInfo(t *testing.T) {

	tests := []struct {
		name string

		agent            *amagent.Agent
		billingAccountID uuid.UUID
		paymentType      bmaccount.PaymentType
		paymentMethod    bmaccount.PaymentMethod

		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.Account
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			billingAccountID: uuid.FromStringOrNil("0a0fc97c-4cdc-11ee-ac88-130f1afddcfa"),
			paymentType:      bmaccount.PaymentTypePrepaid,
			paymentMethod:    bmaccount.PaymentMethodCreditCard,

			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0a0fc97c-4cdc-11ee-ac88-130f1afddcfa"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},
			expectRes: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0a0fc97c-4cdc-11ee-ac88-130f1afddcfa"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
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

			mockReq.EXPECT().BillingV1AccountUpdatePaymentInfo(ctx, tt.billingAccountID, tt.paymentType, tt.paymentMethod).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountUpdatePaymentInfo(ctx, tt.agent, tt.billingAccountID, tt.paymentType, tt.paymentMethod)
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

		agent     *amagent.Agent
		accountID uuid.UUID
		balance   int64

		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.Account
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			accountID: uuid.FromStringOrNil("55867314-4cd8-11ee-b465-73c0486f35ff"),
			balance:   32210000,

			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("650daee2-1060-11ee-aac3-a3c291ad39f5"),
				},
			},
			expectRes: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("650daee2-1060-11ee-aac3-a3c291ad39f5"),
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

			mockReq.EXPECT().BillingV1AccountAddBalanceForce(ctx, tt.accountID, tt.balance).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountAddBalanceForce(ctx, tt.agent, tt.accountID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountUpdateBasicInfo_NoPermission(t *testing.T) {

	tests := []struct {
		name string

		agent            *amagent.Agent
		billingAccountID uuid.UUID
	}{
		{
			name: "customer admin has no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			billingAccountID: uuid.FromStringOrNil("91aea826-4cdc-11ee-9e0f-7bde2e963cc8"),
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

			_, err := h.BillingAccountUpdateBasicInfo(ctx, tt.agent, tt.billingAccountID, "test", "test")
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountUpdatePaymentInfo_NoPermission(t *testing.T) {

	tests := []struct {
		name string

		agent            *amagent.Agent
		billingAccountID uuid.UUID
	}{
		{
			name: "customer admin has no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			billingAccountID: uuid.FromStringOrNil("0a0fc97c-4cdc-11ee-ac88-130f1afddcfa"),
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

			_, err := h.BillingAccountUpdatePaymentInfo(ctx, tt.agent, tt.billingAccountID, bmaccount.PaymentTypePrepaid, bmaccount.PaymentMethodCreditCard)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountSubtractBalanceForce(t *testing.T) {

	tests := []struct {
		name string

		agent     *amagent.Agent
		accountID uuid.UUID
		balance   int64

		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.Account
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			accountID: uuid.FromStringOrNil("55867314-4cd8-11ee-b465-73c0486f35ff"),
			balance:   10000000,

			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("650daee2-1060-11ee-aac3-a3c291ad39f5"),
				},
			},
			expectRes: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("650daee2-1060-11ee-aac3-a3c291ad39f5"),
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

			mockReq.EXPECT().BillingV1AccountSubtractBalanceForce(ctx, tt.accountID, tt.balance).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountSubtractBalanceForce(ctx, tt.agent, tt.accountID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountList_NoPermission(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
	}{
		{
			name: "customer admin has no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
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

			_, err := h.BillingAccountList(ctx, tt.agent, 10, "2020-09-20T03:23:20.995000Z", nil)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountSelfGet(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		responseCustomer       *cscustomer.Customer
		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.WebhookMessage
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

			responseCustomer: &cscustomer.Customer{
				ID:               uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				BillingAccountID: uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
			},
			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().BillingV1AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountSelfGet(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountSelfGet_NoBillingAccount(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		responseCustomer *cscustomer.Customer
	}{
		{
			name: "no billing account",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			responseCustomer: &cscustomer.Customer{
				ID:               uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				BillingAccountID: uuid.Nil,
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)

			_, err := h.BillingAccountSelfGet(ctx, tt.agent)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountSelfGet_NoPermission(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
	}{
		{
			name: "no permission",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: 0,
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

			_, err := h.BillingAccountSelfGet(ctx, tt.agent)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountSelfGet_CustomerNotFound(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
	}{
		{
			name: "customer not found",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(nil, fmt.Errorf("not found"))

			_, err := h.BillingAccountSelfGet(ctx, tt.agent)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountSelfUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		agent  *amagent.Agent
		baName string
		detail string

		responseCustomer       *cscustomer.Customer
		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.WebhookMessage
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
			baName: "test name",
			detail: "test detail",

			responseCustomer: &cscustomer.Customer{
				ID:               uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				BillingAccountID: uuid.FromStringOrNil("91aea826-4cdc-11ee-9e0f-7bde2e963cc8"),
			},
			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91aea826-4cdc-11ee-9e0f-7bde2e963cc8"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91aea826-4cdc-11ee-9e0f-7bde2e963cc8"),
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().BillingV1AccountUpdateBasicInfo(ctx, tt.responseCustomer.BillingAccountID, tt.baName, tt.detail).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountSelfUpdateBasicInfo(ctx, tt.agent, tt.baName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountSelfUpdateBasicInfo_NoBillingAccount(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		responseCustomer *cscustomer.Customer
	}{
		{
			name: "no billing account",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			responseCustomer: &cscustomer.Customer{
				ID:               uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				BillingAccountID: uuid.Nil,
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)

			_, err := h.BillingAccountSelfUpdateBasicInfo(ctx, tt.agent, "test", "test")
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountSelfUpdatePaymentInfo(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		paymentType   bmaccount.PaymentType
		paymentMethod bmaccount.PaymentMethod

		responseCustomer       *cscustomer.Customer
		responseBillingAccount *bmaccount.Account
		expectRes              *bmaccount.WebhookMessage
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
			paymentType:   bmaccount.PaymentTypePrepaid,
			paymentMethod: bmaccount.PaymentMethodCreditCard,

			responseCustomer: &cscustomer.Customer{
				ID:               uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				BillingAccountID: uuid.FromStringOrNil("0a0fc97c-4cdc-11ee-ac88-130f1afddcfa"),
			},
			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0a0fc97c-4cdc-11ee-ac88-130f1afddcfa"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0a0fc97c-4cdc-11ee-ac88-130f1afddcfa"),
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().BillingV1AccountUpdatePaymentInfo(ctx, tt.responseCustomer.BillingAccountID, tt.paymentType, tt.paymentMethod).Return(tt.responseBillingAccount, nil)

			res, err := h.BillingAccountSelfUpdatePaymentInfo(ctx, tt.agent, tt.paymentType, tt.paymentMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingAccountSelfUpdatePaymentInfo_NoBillingAccount(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		responseCustomer *cscustomer.Customer
	}{
		{
			name: "no billing account",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			responseCustomer: &cscustomer.Customer{
				ID:               uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				BillingAccountID: uuid.Nil,
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

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)

			_, err := h.BillingAccountSelfUpdatePaymentInfo(ctx, tt.agent, bmaccount.PaymentTypePrepaid, bmaccount.PaymentMethodCreditCard)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingAccountList(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		size          uint64
		token         string
		filters       map[string]string
		expectFilters map[bmaccount.Field]any

		responseBillingAccounts []bmaccount.Account
		expectRes               []*bmaccount.Account
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},

			10,
			"2020-09-20T03:23:20.995000Z",
			map[string]string{
				"deleted": "false",
			},
			map[bmaccount.Field]any{
				bmaccount.FieldDeleted: false,
			},

			[]bmaccount.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
				},
			},
			[]*bmaccount.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().BillingV1AccountGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseBillingAccounts, nil)

			res, err := h.BillingAccountList(ctx, tt.agent, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

