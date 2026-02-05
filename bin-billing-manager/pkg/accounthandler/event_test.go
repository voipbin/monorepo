package accounthandler

import (
	"context"
	"testing"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
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

		expectFilters map[account.Field]any
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
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d7c03c9c-0e68-11ee-a061-7ff3a502f79b"),
					},
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d7e88e72-0e68-11ee-8ecd-fffa5f8b006c"),
					},
					TMDelete: nil,
				},
			},

			expectFilters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("92435eb0-0e68-11ee-b841-3fe2d10a7ab9"),
				account.FieldDeleted:    false,
			},
			expectRes: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d7c03c9c-0e68-11ee-a061-7ff3a502f79b"),
					},
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d7e88e72-0e68-11ee-8ecd-fffa5f8b006c"),
					},
					TMDelete: nil,
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

			mockDB.EXPECT().AccountList(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseAccounts, nil)

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

func Test_EventCUCustomerCreated(t *testing.T) {

	type test struct {
		name string

		customer *cucustomer.Customer

		responseUUID    uuid.UUID
		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("b4a076de-c8eb-11ef-b239-3f082786094e"),
			},

			responseUUID: uuid.FromStringOrNil("b4a076de-c8eb-11ef-b239-3f082786094e"),
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b4a076de-c8eb-11ef-b239-3f082786094e"),
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
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().AccountCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseUUID).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, tt.responseAccount)

			mockReq.EXPECT().CustomerV1CustomerUpdateBillingAccountID(ctx, tt.customer.ID, tt.responseAccount.ID).Return(tt.customer, nil)

			if err := h.EventCUCustomerCreated(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
