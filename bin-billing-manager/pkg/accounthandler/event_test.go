package accounthandler

import (
	"context"
	"fmt"
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

func Test_EventCUCustomerCreated_error(t *testing.T) {

	type test struct {
		name string

		customer *cucustomer.Customer

		responseUUID    uuid.UUID
		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "create account error",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("d1e1f101-c9eb-11ef-b340-4f193897195f"),
			},

			responseUUID:    uuid.FromStringOrNil("d1e1f101-c9eb-11ef-b340-4f193897195f"),
			responseAccount: nil,
		},
		{
			name: "update customer error",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("d2e2f202-c9eb-11ef-b341-5f294898296g"),
			},

			responseUUID: uuid.FromStringOrNil("d2e2f202-c9eb-11ef-b341-5f294898296g"),
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d2e2f202-c9eb-11ef-b341-5f294898296g"),
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

			if tt.name == "create account error" {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
				mockDB.EXPECT().AccountCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.responseUUID).Return(nil, fmt.Errorf("get failed after create"))

				err := h.EventCUCustomerCreated(ctx, tt.customer)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.name == "update customer error" {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
				mockDB.EXPECT().AccountCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.responseUUID).Return(tt.responseAccount, nil)
				mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, tt.responseAccount)

				mockReq.EXPECT().CustomerV1CustomerUpdateBillingAccountID(ctx, tt.customer.ID, tt.responseAccount.ID).Return(nil, fmt.Errorf("update failed"))

				err := h.EventCUCustomerCreated(ctx, tt.customer)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			}
		})
	}
}

func Test_EventCUCustomerDeleted_error(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		customer   *cucustomer.Customer

		expectFilters map[account.Field]any
	}

	tests := []test{
		{
			name: "list error",

			customerID: uuid.FromStringOrNil("e3f3a313-0e69-11ee-b842-4ff4e11b8ac0"),
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("e3f3a313-0e69-11ee-b842-4ff4e11b8ac0"),
			},

			expectFilters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("e3f3a313-0e69-11ee-b842-4ff4e11b8ac0"),
				account.FieldDeleted:    false,
			},
		},
		{
			name: "empty accounts",

			customerID: uuid.FromStringOrNil("e4f4b424-0e69-11ee-b843-5005f22c9bd1"),
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("e4f4b424-0e69-11ee-b843-5005f22c9bd1"),
			},

			expectFilters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("e4f4b424-0e69-11ee-b843-5005f22c9bd1"),
				account.FieldDeleted:    false,
			},
		},
		{
			name: "delete error continues",

			customerID: uuid.FromStringOrNil("e5f5c535-0e69-11ee-b844-6116033dace2"),
			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("e5f5c535-0e69-11ee-b844-6116033dace2"),
			},

			expectFilters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("e5f5c535-0e69-11ee-b844-6116033dace2"),
				account.FieldDeleted:    false,
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

			if tt.name == "list error" {
				mockDB.EXPECT().AccountList(ctx, uint64(1000), "", tt.expectFilters).Return(nil, fmt.Errorf("list failed"))

				err := h.EventCUCustomerDeleted(ctx, tt.customer)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.name == "empty accounts" {
				mockDB.EXPECT().AccountList(ctx, uint64(1000), "", tt.expectFilters).Return([]*account.Account{}, nil)

				err := h.EventCUCustomerDeleted(ctx, tt.customer)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}

			if tt.name == "delete error continues" {
				accounts := []*account.Account{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("f6060d46-0e69-11ee-b845-7227144ebdf3"),
						},
					},
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("f7071e57-0e69-11ee-b846-8338255fceg4"),
						},
					},
				}

				mockDB.EXPECT().AccountList(ctx, uint64(1000), "", tt.expectFilters).Return(accounts, nil)

				// First delete fails
				mockDB.EXPECT().AccountDelete(ctx, accounts[0].ID).Return(fmt.Errorf("delete failed"))

				// Second delete succeeds
				mockDB.EXPECT().AccountDelete(ctx, accounts[1].ID).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, accounts[1].ID).Return(accounts[1], nil)

				err := h.EventCUCustomerDeleted(ctx, tt.customer)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}
		})
	}
}
