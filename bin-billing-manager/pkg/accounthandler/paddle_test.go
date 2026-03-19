package accounthandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_PaddleCreditTopUp(t *testing.T) {
	tests := []struct {
		name string

		customerID         uuid.UUID
		amountCreditMicros int64
		eventID            string

		// idempotency check
		responseIdempotencyErr error

		// GetByCustomerID mocks
		responseCustomer    *cmcustomer.Customer
		responseCustomerErr error
		responseAccount     *account.Account

		// DB mock
		responseAddCreditErr error

		expectErr bool
	}{
		{
			name:               "normal",
			customerID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
			amountCreditMicros: 10000000,
			eventID:            "evt_credit_001",

			responseIdempotencyErr: dbhandler.ErrNotFound,

			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
				BillingAccountID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
				},
			},

			responseAddCreditErr: nil,
			expectErr:            false,
		},
		{
			name:               "idempotent - already processed",
			customerID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
			amountCreditMicros: 10000000,
			eventID:            "evt_credit_dup",

			responseIdempotencyErr: nil, // record found → already processed

			expectErr: false,
		},
		{
			name:               "invalid amount - zero",
			customerID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
			amountCreditMicros: 0,
			eventID:            "evt_credit_zero",

			expectErr: true,
		},
		{
			name:               "invalid amount - negative",
			customerID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
			amountCreditMicros: -5000000,
			eventID:            "evt_credit_neg",

			expectErr: true,
		},
		{
			name:               "customer not found",
			customerID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000099"),
			amountCreditMicros: 10000000,
			eventID:            "evt_credit_no_cust",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseCustomerErr:    fmt.Errorf("customer not found"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := accountHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, tt.eventID)

			// Amount guard short-circuits before any DB calls
			if tt.amountCreditMicros > 0 {
				// Idempotency check
				mockDB.EXPECT().BillingGetByIdempotencyKey(ctx, idempotencyKey).Return(&billing.Billing{}, tt.responseIdempotencyErr)

				if tt.responseIdempotencyErr == dbhandler.ErrNotFound {
					// GetByCustomerID chain
					if tt.responseCustomerErr != nil {
						mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(nil, tt.responseCustomerErr)
					} else {
						mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
						mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)

						// Atomic add credit
						mockDB.EXPECT().AccountPaddleAddCredit(ctx, tt.responseAccount.ID, tt.amountCreditMicros, tt.responseAccount.CustomerID, idempotencyKey).Return(tt.responseAddCreditErr)
					}
				}
			}

			err := h.PaddleCreditTopUp(ctx, tt.customerID, tt.amountCreditMicros, tt.eventID)
			if (err != nil) != tt.expectErr {
				t.Errorf("PaddleCreditTopUp() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func Test_PaddleSubscriptionCreate(t *testing.T) {
	tests := []struct {
		name string

		customerID   uuid.UUID
		planType     account.PlanType
		paddleSubID  string
		paddleCustID string
		eventID      string

		responseIdempotencyErr   error
		responseCustomer         *cmcustomer.Customer
		responseCustomerErr      error
		responseAccount          *account.Account
		responseAccountUpdateErr error

		expectErr bool
	}{
		{
			name:         "normal - basic plan",
			customerID:   uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
			planType:     account.PlanTypeBasic,
			paddleSubID:  "sub_abc123",
			paddleCustID: "ctm_def456",
			eventID:      "evt_sub_create_001",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
				BillingAccountID: uuid.FromStringOrNil("b0000002-0000-0000-0000-000000000001"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000002-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
				},
			},

			expectErr: false,
		},
		{
			name:         "idempotent - already processed",
			customerID:   uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
			planType:     account.PlanTypeBasic,
			paddleSubID:  "sub_abc123",
			paddleCustID: "ctm_def456",
			eventID:      "evt_sub_create_dup",

			responseIdempotencyErr: nil, // record found → already processed

			expectErr: false,
		},
		{
			name:         "customer not found",
			customerID:   uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000099"),
			planType:     account.PlanTypeBasic,
			paddleSubID:  "sub_abc123",
			paddleCustID: "ctm_def456",
			eventID:      "evt_sub_create_no_cust",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseCustomerErr:    fmt.Errorf("customer not found"),

			expectErr: true,
		},
		{
			name:         "account update paddle IDs fails",
			customerID:   uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
			planType:     account.PlanTypeBasic,
			paddleSubID:  "sub_abc123",
			paddleCustID: "ctm_def456",
			eventID:      "evt_sub_create_update_fail",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
				BillingAccountID: uuid.FromStringOrNil("b0000002-0000-0000-0000-000000000001"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000002-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000001"),
				},
			},
			responseAccountUpdateErr: fmt.Errorf("db update error"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := accountHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, tt.eventID)

			mockDB.EXPECT().BillingGetByIdempotencyKey(ctx, idempotencyKey).Return(&billing.Billing{}, tt.responseIdempotencyErr)

			if tt.responseIdempotencyErr == dbhandler.ErrNotFound {
				if tt.responseCustomerErr != nil {
					mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(nil, tt.responseCustomerErr)
				} else {
					mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
					mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)

					// Store paddle IDs
					mockDB.EXPECT().AccountUpdate(ctx, tt.responseAccount.ID, map[account.Field]any{
						account.FieldPaddleSubscriptionID: tt.paddleSubID,
						account.FieldPaddleCustomerID:     tt.paddleCustID,
					}).Return(tt.responseAccountUpdateErr)

					if tt.responseAccountUpdateErr == nil {
						// UpdatePlanType chain
						mockDB.EXPECT().AccountUpdate(ctx, tt.responseAccount.ID, map[account.Field]any{
							account.FieldPlanType: tt.planType,
						}).Return(nil)
						mockDB.EXPECT().AccountGet(ctx, tt.responseAccount.ID).Return(tt.responseAccount, nil)

						// Token top-up
						tokenAllowance := account.PlanTokenMap[tt.planType]
						mockDB.EXPECT().AccountPaddleTopUpTokens(ctx, tt.responseAccount.ID, tt.responseAccount.CustomerID, tokenAllowance, string(tt.planType), billing.TransactionTypeTopUp, idempotencyKey).Return(nil)
					}
				}
			}

			err := h.PaddleSubscriptionCreate(ctx, tt.customerID, tt.planType, tt.paddleSubID, tt.paddleCustID, tt.eventID)
			if (err != nil) != tt.expectErr {
				t.Errorf("PaddleSubscriptionCreate() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func Test_PaddleSubscriptionUpdate(t *testing.T) {
	tests := []struct {
		name string

		paddleSubID string
		newPlanType account.PlanType
		eventID     string

		responseIdempotencyErr error
		responseAccount        *account.Account

		expectErr bool
	}{
		{
			name:        "normal - upgrade to professional",
			paddleSubID: "sub_update_001",
			newPlanType: account.PlanTypeProfessional,
			eventID:     "evt_sub_update_001",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000008-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000008-0000-0000-0000-000000000001"),
				},
				PlanType: account.PlanTypeBasic,
			},

			expectErr: false,
		},
		{
			name:        "idempotent - already processed",
			paddleSubID: "sub_update_002",
			newPlanType: account.PlanTypeProfessional,
			eventID:     "evt_sub_update_dup",

			responseIdempotencyErr: nil, // record found → already processed

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := accountHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, tt.eventID)

			mockDB.EXPECT().BillingGetByIdempotencyKey(ctx, idempotencyKey).Return(&billing.Billing{}, tt.responseIdempotencyErr)

			if tt.responseIdempotencyErr == dbhandler.ErrNotFound {
				mockDB.EXPECT().AccountGetByPaddleSubscriptionID(ctx, tt.paddleSubID).Return(tt.responseAccount, nil)

				// UpdatePlanType chain
				mockDB.EXPECT().AccountUpdate(ctx, tt.responseAccount.ID, map[account.Field]any{
					account.FieldPlanType: tt.newPlanType,
				}).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.responseAccount.ID).Return(tt.responseAccount, nil)

				// Token reset to new plan allowance
				tokenAllowance := account.PlanTokenMap[tt.newPlanType]
				mockDB.EXPECT().AccountPaddleTopUpTokens(ctx, tt.responseAccount.ID, tt.responseAccount.CustomerID, tokenAllowance, string(tt.newPlanType), billing.TransactionTypeAdjustment, idempotencyKey).Return(nil)
			}

			err := h.PaddleSubscriptionUpdate(ctx, tt.paddleSubID, tt.newPlanType, tt.eventID)
			if (err != nil) != tt.expectErr {
				t.Errorf("PaddleSubscriptionUpdate() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func Test_PaddleSubscriptionCancel(t *testing.T) {
	tests := []struct {
		name string

		paddleSubID string
		eventID     string

		responseIdempotencyErr error
		responseAccount        *account.Account

		expectErr bool
	}{
		{
			name:        "normal - cancel to free",
			paddleSubID: "sub_cancel_001",
			eventID:     "evt_sub_cancel_001",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000003-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000003-0000-0000-0000-000000000001"),
				},
				PlanType: account.PlanTypeBasic,
			},

			expectErr: false,
		},
		{
			name:        "idempotent - already processed",
			paddleSubID: "sub_cancel_002",
			eventID:     "evt_sub_cancel_dup",

			responseIdempotencyErr: nil, // record found → already processed

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := accountHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, tt.eventID)

			mockDB.EXPECT().BillingGetByIdempotencyKey(ctx, idempotencyKey).Return(&billing.Billing{}, tt.responseIdempotencyErr)

			if tt.responseIdempotencyErr == dbhandler.ErrNotFound {
				mockDB.EXPECT().AccountGetByPaddleSubscriptionID(ctx, tt.paddleSubID).Return(tt.responseAccount, nil)

				// UpdatePlanType to Free
				mockDB.EXPECT().AccountUpdate(ctx, tt.responseAccount.ID, map[account.Field]any{
					account.FieldPlanType: account.PlanTypeFree,
				}).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.responseAccount.ID).Return(tt.responseAccount, nil)

				// Token reset to free allowance
				tokenAllowance := account.PlanTokenMap[account.PlanTypeFree]
				mockDB.EXPECT().AccountPaddleTopUpTokens(ctx, tt.responseAccount.ID, tt.responseAccount.CustomerID, tokenAllowance, string(account.PlanTypeFree), billing.TransactionTypeAdjustment, idempotencyKey).Return(nil)
			}

			err := h.PaddleSubscriptionCancel(ctx, tt.paddleSubID, tt.eventID)
			if (err != nil) != tt.expectErr {
				t.Errorf("PaddleSubscriptionCancel() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func Test_PaddleSubscriptionRenew(t *testing.T) {
	tests := []struct {
		name string

		paddleSubID string
		eventID     string

		responseIdempotencyErr error
		responseAccount        *account.Account

		expectErr bool
	}{
		{
			name:        "normal - renew basic",
			paddleSubID: "sub_renew_001",
			eventID:     "evt_sub_renew_001",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000004-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000004-0000-0000-0000-000000000001"),
				},
				PlanType: account.PlanTypeBasic,
			},

			expectErr: false,
		},
		{
			name:        "skip - free plan (post-cancellation)",
			paddleSubID: "sub_renew_002",
			eventID:     "evt_sub_renew_002",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000005-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000005-0000-0000-0000-000000000001"),
				},
				PlanType: account.PlanTypeFree,
			},

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := accountHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, tt.eventID)

			mockDB.EXPECT().BillingGetByIdempotencyKey(ctx, idempotencyKey).Return(&billing.Billing{}, tt.responseIdempotencyErr)

			if tt.responseIdempotencyErr == dbhandler.ErrNotFound {
				mockDB.EXPECT().AccountGetByPaddleSubscriptionID(ctx, tt.paddleSubID).Return(tt.responseAccount, nil)

				if tt.responseAccount.PlanType != account.PlanTypeFree {
					tokenAllowance := account.PlanTokenMap[tt.responseAccount.PlanType]
					mockDB.EXPECT().AccountPaddleTopUpTokens(ctx, tt.responseAccount.ID, tt.responseAccount.CustomerID, tokenAllowance, string(tt.responseAccount.PlanType), billing.TransactionTypeTopUp, idempotencyKey).Return(nil)
				}
				// free plan: no mock expectations → skips renewal
			}

			err := h.PaddleSubscriptionRenew(ctx, tt.paddleSubID, tt.eventID)
			if (err != nil) != tt.expectErr {
				t.Errorf("PaddleSubscriptionRenew() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func Test_PaddleRefund(t *testing.T) {
	tests := []struct {
		name string

		customerID         uuid.UUID
		amountCreditMicros int64
		eventID            string

		responseIdempotencyErr error
		responseCustomer       *cmcustomer.Customer
		responseAccount        *account.Account
		responseUpdatedAccount *account.Account

		expectErr bool
	}{
		{
			name:               "normal - positive balance after refund",
			customerID:         uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
			amountCreditMicros: 5000000,
			eventID:            "evt_refund_001",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
				BillingAccountID: uuid.FromStringOrNil("b0000006-0000-0000-0000-000000000001"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000006-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
				},
				BalanceCredit: 10000000,
			},
			responseUpdatedAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000006-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
				},
				BalanceCredit: 5000000, // Still positive
			},

			expectErr: false,
		},
		{
			name:               "negative balance - account frozen",
			customerID:         uuid.FromStringOrNil("a0000007-0000-0000-0000-000000000001"),
			amountCreditMicros: 15000000,
			eventID:            "evt_refund_002",

			responseIdempotencyErr: dbhandler.ErrNotFound,
			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("a0000007-0000-0000-0000-000000000001"),
				BillingAccountID: uuid.FromStringOrNil("b0000007-0000-0000-0000-000000000001"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000007-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000007-0000-0000-0000-000000000001"),
				},
				BalanceCredit: 10000000,
			},
			responseUpdatedAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000007-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000007-0000-0000-0000-000000000001"),
				},
				BalanceCredit: -5000000, // Negative → freeze
			},

			expectErr: false,
		},
		{
			name:               "invalid amount - zero",
			customerID:         uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
			amountCreditMicros: 0,
			eventID:            "evt_refund_zero",

			expectErr: true,
		},
		{
			name:               "invalid amount - negative",
			customerID:         uuid.FromStringOrNil("a0000006-0000-0000-0000-000000000001"),
			amountCreditMicros: -1000000,
			eventID:            "evt_refund_neg",

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := accountHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, tt.eventID)

			// Amount guard short-circuits before any DB calls
			if tt.amountCreditMicros > 0 {
				mockDB.EXPECT().BillingGetByIdempotencyKey(ctx, idempotencyKey).Return(&billing.Billing{}, tt.responseIdempotencyErr)

				if tt.responseIdempotencyErr == dbhandler.ErrNotFound {
					// GetByCustomerID chain
					mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
					mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)

					// Atomic subtract credit
					mockDB.EXPECT().AccountPaddleSubtractCredit(ctx, tt.responseAccount.ID, tt.amountCreditMicros, tt.responseAccount.CustomerID, idempotencyKey).Return(nil)

					// Get updated account for freeze check
					mockDB.EXPECT().AccountGet(ctx, tt.responseAccount.ID).Return(tt.responseUpdatedAccount, nil)

					// Freeze if negative
					if tt.responseUpdatedAccount.BalanceCredit < 0 {
						mockDB.EXPECT().AccountSetStatus(ctx, tt.responseAccount.ID, account.StatusFrozen).Return(nil)
						mockDB.EXPECT().AccountGet(ctx, tt.responseAccount.ID).Return(tt.responseUpdatedAccount, nil)
						mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountUpdated, tt.responseUpdatedAccount)
					}
				}
			}

			err := h.PaddleRefund(ctx, tt.customerID, tt.amountCreditMicros, tt.eventID)
			if (err != nil) != tt.expectErr {
				t.Errorf("PaddleRefund() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
