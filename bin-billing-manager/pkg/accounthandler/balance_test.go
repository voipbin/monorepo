package accounthandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/allowancehandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_IsValidBalanceByCustomerID(t *testing.T) {

	type test struct {
		name string

		customerID  uuid.UUID
		billingType billing.ReferenceType
		country     string
		count       int

		responseCustomer *cmcustomer.Customer
		responseAccount  *account.Account
		expectRes        bool
	}

	tests := []test{
		{
			name: "account has enough balance",

			customerID:  uuid.FromStringOrNil("87abc36a-09f5-11ee-9b1c-d3d80f26eacd"),
			billingType: billing.ReferenceTypeCall,
			count:       1,

			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("87abc36a-09f5-11ee-9b1c-d3d80f26eacd"),
				BillingAccountID: uuid.FromStringOrNil("8802cee4-09f5-11ee-9491-378bbbda7e50"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8802cee4-09f5-11ee-9491-378bbbda7e50"),
				},
				Balance:  10.0,
				TMDelete: nil,
			},
			expectRes: true,
		},
		{
			name: "account has no balance but the plan type is unlimited",

			customerID:  uuid.FromStringOrNil("a0c6f8c4-09f5-11ee-8ad8-4f77e9290706"),
			billingType: billing.ReferenceTypeNumber,
			count:       1,

			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("a0c6f8c4-09f5-11ee-8ad8-4f77e9290706"),
				BillingAccountID: uuid.FromStringOrNil("a0e702cc-09f5-11ee-a6d0-e77ef7485142"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0e702cc-09f5-11ee-a6d0-e77ef7485142"),
				},
				PlanType: account.PlanTypeUnlimited,
				TMDelete: nil,
			},
			expectRes: true,
		},
		{
			name: "customer not found error",

			customerID:  uuid.FromStringOrNil("b1c1d1e1-1111-11ee-86c6-111111111111"),
			billingType: billing.ReferenceTypeCall,
			count:       1,

			responseCustomer: nil,
			responseAccount:  nil,
			expectRes:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAllowance := allowancehandler.NewMockAllowanceHandler(mc)

			h := accountHandler{
				utilHandler:      mockUtil,
				db:               mockDB,
				notifyHandler:    mockNotify,
				reqHandler:       mockReq,
				allowanceHandler: mockAllowance,
			}
			ctx := context.Background()

			if tt.name == "customer not found error" {
				mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(nil, fmt.Errorf("customer not found"))

				_, err := h.IsValidBalanceByCustomerID(ctx, tt.customerID, tt.billingType, tt.country, tt.count)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)

			// IsValidBalance will call AccountGet again
			mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)

			// For "account has enough balance" test, IsValidBalance checks allowance then credit
			if tt.name == "account has enough balance" {
				// Call type: check allowance first, then fall back to credit
				mockAllowance.EXPECT().GetCurrentCycle(ctx, tt.responseAccount.ID).Return(nil, dbhandler.ErrNotFound)
			}

			res, err := h.IsValidBalanceByCustomerID(ctx, tt.customerID, tt.billingType, tt.country, tt.count)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_IsValidBalance(t *testing.T) {

	type test struct {
		name string

		accountID   uuid.UUID
		billingType billing.ReferenceType
		country     string
		count       int

		responseAccount *account.Account
		expectRes       bool
		expectErr       bool
	}

	tmDelete := time.Now()

	tests := []test{
		{
			name: "account has enough balance",

			accountID:   uuid.FromStringOrNil("u53d1e596-1342-11ee-86c6-23afd019902d"),
			billingType: billing.ReferenceTypeCall,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("u53d1e596-1342-11ee-86c6-23afd019902d"),
				},
				Balance:  10.0,
				TMDelete: nil,
			},
			expectRes: true,
		},
		{
			name: "account has no balance but the plan type is unlimited",

			accountID:   uuid.FromStringOrNil("540fbd80-1342-11ee-bdfc-e3cff38ae489"),
			billingType: billing.ReferenceTypeNumber,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("540fbd80-1342-11ee-bdfc-e3cff38ae489"),
				},
				PlanType: account.PlanTypeUnlimited,
				TMDelete: nil,
			},
			expectRes: true,
		},
		{
			name: "deleted account returns false",

			accountID:   uuid.FromStringOrNil("a1b1c1d1-1111-11ee-86c6-111111111111"),
			billingType: billing.ReferenceTypeCall,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b1c1d1-1111-11ee-86c6-111111111111"),
				},
				Balance:  10.0,
				TMDelete: &tmDelete,
			},
			expectRes: false,
		},
		{
			name: "insufficient balance for call",

			accountID:   uuid.FromStringOrNil("a2b2c2d2-2222-11ee-86c6-222222222222"),
			billingType: billing.ReferenceTypeCall,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a2b2c2d2-2222-11ee-86c6-222222222222"),
				},
				Balance:  0.001,
				TMDelete: nil,
			},
			expectRes: false,
		},
		{
			name: "insufficient balance for number",

			accountID:   uuid.FromStringOrNil("a3b3c3d3-3333-11ee-86c6-333333333333"),
			billingType: billing.ReferenceTypeNumber,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a3b3c3d3-3333-11ee-86c6-333333333333"),
				},
				Balance:  4.99,
				TMDelete: nil,
			},
			expectRes: false,
		},
		{
			name: "call_extension always valid regardless of balance",

			accountID:   uuid.FromStringOrNil("a4b4c4d4-4444-11ee-86c6-444444444444"),
			billingType: billing.ReferenceTypeCallExtension,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4b4c4d4-4444-11ee-86c6-444444444444"),
				},
				Balance:  0,
				TMDelete: nil,
			},
			expectRes: true,
		},
		{
			name: "sms with enough balance",

			accountID:   uuid.FromStringOrNil("a5b5c5d5-5555-11ee-86c6-555555555555"),
			billingType: billing.ReferenceTypeSMS,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a5b5c5d5-5555-11ee-86c6-555555555555"),
				},
				Balance:  1.0,
				TMDelete: nil,
			},
			expectRes: true,
		},
		{
			name: "count less than 1 normalized to 1",

			accountID:   uuid.FromStringOrNil("a6b6c6d6-6666-11ee-86c6-666666666666"),
			billingType: billing.ReferenceTypeCall,
			count:       0,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6b6c6d6-6666-11ee-86c6-666666666666"),
				},
				Balance:  10.0,
				TMDelete: nil,
			},
			expectRes: true,
		},
		{
			name: "unsupported billing type returns error",

			accountID:   uuid.FromStringOrNil("a7b7c7d7-7777-11ee-86c6-777777777777"),
			billingType: billing.ReferenceType("unknown"),
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a7b7c7d7-7777-11ee-86c6-777777777777"),
				},
				Balance:  10.0,
				TMDelete: nil,
			},
			expectRes: false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAllowance := allowancehandler.NewMockAllowanceHandler(mc)

			h := accountHandler{
				utilHandler:      mockUtil,
				db:               mockDB,
				notifyHandler:    mockNotify,
				reqHandler:       mockReq,
				allowanceHandler: mockAllowance,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			// Set up allowance mocks for types that check allowance
			if tt.responseAccount.TMDelete == nil && tt.responseAccount.PlanType != account.PlanTypeUnlimited && tt.billingType != billing.ReferenceTypeCallExtension && !tt.expectErr {
				switch tt.billingType {
				case billing.ReferenceTypeCall:
					// Call checks allowance first
					mockAllowance.EXPECT().GetCurrentCycle(ctx, tt.accountID).Return(nil, dbhandler.ErrNotFound)
				case billing.ReferenceTypeSMS:
					// SMS checks allowance first
					mockAllowance.EXPECT().GetCurrentCycle(ctx, tt.accountID).Return(nil, dbhandler.ErrNotFound)
				}
			}

			res, err := h.IsValidBalance(ctx, tt.accountID, tt.billingType, tt.country, tt.count)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
