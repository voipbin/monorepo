package accounthandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/pkg/dbhandler"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_EventCustomerCreated(t *testing.T) {

	tests := []struct {
		name string

		customer *cucustomer.Customer

		responseAccounts *account.Account
		responseUUID     uuid.UUID
		expectFilters    map[string]string
		expectAccount    *account.Account
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("f02bb3b6-1b6d-11ef-9166-b349a9bf3799"),
			},

			responseAccounts: &account.Account{
				ID:       uuid.FromStringOrNil("f055b6ac-1b6d-11ef-9381-033d27b16b0d"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseUUID: uuid.FromStringOrNil("f0aaee06-1b6d-11ef-acb1-87b1e9c73a2f"),

			expectFilters: map[string]string{
				"customer_id": "f02bb3b6-1b6d-11ef-9166-b349a9bf3799",
				"deleted":     "false",
			},
			expectAccount: &account.Account{
				ID:         uuid.FromStringOrNil("f0aaee06-1b6d-11ef-acb1-87b1e9c73a2f"),
				CustomerID: uuid.FromStringOrNil("f02bb3b6-1b6d-11ef-9166-b349a9bf3799"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &accountHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()
			mockDB.EXPECT().AccountGets(ctx, gomock.Any(), uint64(1), tt.expectFilters).Return([]*account.Account{}, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().AccountCreate(ctx, tt.expectAccount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseUUID).Return(tt.expectAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, tt.expectAccount)

			if err := h.EventCustomerCreated(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer *cucustomer.Customer

		expectFilters    map[string]string
		responseAccounts []*account.Account
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("53812672-1b6d-11ef-9390-7bcc54eaeb10"),
			},

			expectFilters: map[string]string{
				"customer_id": "53812672-1b6d-11ef-9390-7bcc54eaeb10",
				"deleted":     "false",
			},
			responseAccounts: []*account.Account{
				{
					ID:       uuid.FromStringOrNil("53b343d2-1b6d-11ef-8d3f-87f6aa5d9616"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &accountHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()
			mockDB.EXPECT().AccountGets(ctx, gomock.Any(), uint64(1), tt.expectFilters).Return(tt.responseAccounts, nil)

			// delete
			for _, f := range tt.responseAccounts {

				mockDB.EXPECT().AccountDelete(ctx, f.ID).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, f.ID).Return(f, nil)
				mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountDeleted, f)
			}

			if err := h.EventCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
