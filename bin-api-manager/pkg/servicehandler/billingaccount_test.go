package servicehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	bmaccount "monorepo/bin-billing-manager/models/account"
	bmallowance "monorepo/bin-billing-manager/models/allowance"
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
		balance   float32

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
			balance:   32.21,

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

func Test_BillingAccountAllowancesGet(t *testing.T) {
	cycleStart := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	cycleEnd := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		agent            *amagent.Agent
		billingAccountID uuid.UUID
		pageSize         uint64
		pageToken        string

		responseBillingAccount *bmaccount.Account
		responseAllowances     []*bmallowance.Allowance

		expectRes []*bmallowance.WebhookMessage
	}{
		{
			name: "normal with results",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			billingAccountID: uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
			pageSize:         10,
			pageToken:        "",

			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},
			responseAllowances: []*bmallowance.Allowance{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a1b2c3d4-1234-5678-9abc-def012345678"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					AccountID:   uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
					CycleStart:  &cycleStart,
					CycleEnd:    &cycleEnd,
					TokensTotal: 10000,
					TokensUsed:  3500,
					TMCreate:    &tmCreate,
				},
			},

			expectRes: []*bmallowance.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a1b2c3d4-1234-5678-9abc-def012345678"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					AccountID:   uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
					CycleStart:  &cycleStart,
					CycleEnd:    &cycleEnd,
					TokensTotal: 10000,
					TokensUsed:  3500,
					TMCreate:    &tmCreate,
				},
			},
		},
		{
			name: "empty results",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			billingAccountID: uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
			pageSize:         10,
			pageToken:        "",

			responseBillingAccount: &bmaccount.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d18d036a-105b-11ee-9f29-bb51d45198bc"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: nil,
			},
			responseAllowances: []*bmallowance.Allowance{},

			expectRes: []*bmallowance.WebhookMessage{},
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
			mockReq.EXPECT().BillingV1AccountAllowancesGet(ctx, tt.billingAccountID, tt.pageSize, tt.pageToken).Return(tt.responseAllowances, nil)

			res, err := h.BillingAccountAllowancesGet(ctx, tt.agent, tt.billingAccountID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
