package accounthandler

import (
	"context"
	"testing"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		customer   *cucustomer.Customer

		responseAccounts []*account.Account

		expectFilters map[string]string
		expectRes     []*account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("92435eb0-0e68-11ee-b841-3fe2d10a7ab9"),
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("92435eb0-0e68-11ee-b841-3fe2d10a7ab9"),
			},

			responseAccounts: []*account.Account{
				{
					ID:       uuid.FromStringOrNil("d7c03c9c-0e68-11ee-a061-7ff3a502f79b"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("d7e88e72-0e68-11ee-8ecd-fffa5f8b006c"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},

			expectFilters: map[string]string{
				"customer_id": "92435eb0-0e68-11ee-b841-3fe2d10a7ab9",
				"deleted":     "false",
			},
			expectRes: []*account.Account{
				{
					ID:       uuid.FromStringOrNil("d7c03c9c-0e68-11ee-a061-7ff3a502f79b"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("d7e88e72-0e68-11ee-8ecd-fffa5f8b006c"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountGets(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseAccounts, nil)

			for _, a := range tt.responseAccounts {
				mockDB.EXPECT().AccountDelete(ctx, a.ID).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, a.ID).Return(a, nil)
			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
