package customerhandler

import (
	"context"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

func Test_IsValidBalance(t *testing.T) {
	tests := []struct {
		name string

		customerID  uuid.UUID
		billingType bmbilling.ReferenceType
		country     string
		count       int

		responseCustomer *customer.Customer
		responseValid    bool

		expectRes bool
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("7c9e30bc-0f8a-11ee-81e3-4be5aea558dd"),
			billingType: bmbilling.ReferenceTypeCall,
			country:     "us",
			count:       3,

			responseCustomer: &customer.Customer{
				ID:               uuid.FromStringOrNil("7c9e30bc-0f8a-11ee-81e3-4be5aea558dd"),
				BillingAccountID: uuid.FromStringOrNil("7ccb4c96-0f8a-11ee-b0dc-9b9d7bfd6099"),
			},

			responseValid: true,

			expectRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerGet(ctx, tt.customerID.Return(tt.responseCustomer, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalance(ctx, tt.responseCustomer.BillingAccountID, tt.billingType, tt.country, tt.count.Return(tt.responseValid, nil)

			res, err := h.IsValidBalance(ctx, tt.customerID, tt.billingType, tt.country, tt.count)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
