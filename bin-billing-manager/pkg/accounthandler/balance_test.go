package accounthandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
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
				ID:       uuid.FromStringOrNil("8802cee4-09f5-11ee-9491-378bbbda7e50"),
				Balance:  10.0,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			expectRes: true,
		},
		{
			name: "account has no balance but the type is admin",

			customerID:  uuid.FromStringOrNil("a0c6f8c4-09f5-11ee-8ad8-4f77e9290706"),
			billingType: billing.ReferenceTypeNumber,
			count:       1,

			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("a0c6f8c4-09f5-11ee-8ad8-4f77e9290706"),
				BillingAccountID: uuid.FromStringOrNil("a0e702cc-09f5-11ee-a6d0-e77ef7485142"),
			},
			responseAccount: &account.Account{
				ID:       uuid.FromStringOrNil("a0e702cc-09f5-11ee-a6d0-e77ef7485142"),
				Type:     account.TypeAdmin,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			expectRes: true,
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)
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
	}

	tests := []test{
		{
			name: "account has enough balance",

			accountID:   uuid.FromStringOrNil("u53d1e596-1342-11ee-86c6-23afd019902d"),
			billingType: billing.ReferenceTypeCall,
			count:       1,

			responseAccount: &account.Account{
				ID:       uuid.FromStringOrNil("u53d1e596-1342-11ee-86c6-23afd019902d"),
				Balance:  10.0,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			expectRes: true,
		},
		{
			name: "account has no balance but the type is admin",

			accountID:   uuid.FromStringOrNil("540fbd80-1342-11ee-bdfc-e3cff38ae489"),
			billingType: billing.ReferenceTypeNumber,
			count:       1,

			responseAccount: &account.Account{
				ID:       uuid.FromStringOrNil("540fbd80-1342-11ee-bdfc-e3cff38ae489"),
				Type:     account.TypeAdmin,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			expectRes: true,
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			res, err := h.IsValidBalance(ctx, tt.accountID, tt.billingType, tt.country, tt.count)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
