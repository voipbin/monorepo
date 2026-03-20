package servicehandler

import (
	"context"
	"reflect"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

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

func Test_BillingAccountUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		agent              *amagent.Agent
		billingAccountID   uuid.UUID
		billingAccountName string
		detail             string

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
			expectRes: &bmaccount.WebhookMessage{
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

			mockReq.EXPECT().BillingV1AccountGet(ctx, tt.billingAccountID).Return(tt.responseBillingAccount, nil)
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
			expectRes: &bmaccount.WebhookMessage{
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

			mockReq.EXPECT().BillingV1AccountGet(ctx, tt.billingAccountID).Return(tt.responseBillingAccount, nil)
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
		expectRes              *bmaccount.WebhookMessage
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
			expectRes: &bmaccount.WebhookMessage{
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

