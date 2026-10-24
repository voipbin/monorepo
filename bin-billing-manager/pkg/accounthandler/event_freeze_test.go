package accounthandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_EventCUCustomerFrozen(t *testing.T) {

	customerID := uuid.FromStringOrNil("92435eb0-0e68-11ee-b841-3fe2d10a7ab9")
	accountID1 := uuid.FromStringOrNil("d7c03c9c-0e68-11ee-a061-7ff3a502f79b")
	accountID2 := uuid.FromStringOrNil("d7e88e72-0e68-11ee-8ecd-fffa5f8b006c")
	errCause := errors.New("db failure")

	type test struct {
		name string

		responseAccounts []*account.Account
		responseListErr  error
		// error of the status update for the account with the same index
		responseSetErrs []error

		expectErrIs error
	}

	tests := []test{
		{
			name: "two accounts are frozen",

			responseAccounts: []*account.Account{
				{Identity: commonidentity.Identity{ID: accountID1}},
				{Identity: commonidentity.Identity{ID: accountID2}},
			},
			responseSetErrs: []error{nil, nil},
		},
		{
			name: "no account",

			responseAccounts: []*account.Account{},
		},
		{
			name: "a failing account does not stop the next one",

			responseAccounts: []*account.Account{
				{Identity: commonidentity.Identity{ID: accountID1}},
				{Identity: commonidentity.Identity{ID: accountID2}},
			},
			responseSetErrs: []error{errCause, nil},
		},
		{
			name: "list fails and no account is touched",

			responseListErr: errCause,

			expectErrIs: errCause,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &accountHandler{
				db: mockDB,
			}
			ctx := context.Background()

			expectFilters := map[account.Field]any{
				account.FieldCustomerID: customerID,
				account.FieldDeleted:    false,
			}
			mockDB.EXPECT().AccountList(ctx, uint64(1000), "", expectFilters).Return(tt.responseAccounts, tt.responseListErr)

			for i, a := range tt.responseAccounts {
				mockDB.EXPECT().AccountSetStatus(ctx, a.ID, account.StatusFrozen).Return(tt.responseSetErrs[i]).Times(1)
			}

			err := h.EventCUCustomerFrozen(ctx, &cucustomer.Customer{ID: customerID})
			if tt.expectErrIs == nil {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.expectErrIs) {
				t.Errorf("Wrong match. expect: error caused by %v, got: %v", tt.expectErrIs, err)
			}
		})
	}
}

func Test_EventCUCustomerRecovered(t *testing.T) {

	customerID := uuid.FromStringOrNil("92435eb0-0e68-11ee-b841-3fe2d10a7ab9")
	accountID1 := uuid.FromStringOrNil("d7c03c9c-0e68-11ee-a061-7ff3a502f79b")
	accountID2 := uuid.FromStringOrNil("d7e88e72-0e68-11ee-8ecd-fffa5f8b006c")
	errCause := errors.New("db failure")

	type test struct {
		name string

		responseAccounts []*account.Account
		responseListErr  error
		responseSetErrs  []error

		expectErrIs error
	}

	tests := []test{
		{
			name: "two frozen accounts are activated",

			responseAccounts: []*account.Account{
				{Identity: commonidentity.Identity{ID: accountID1}},
				{Identity: commonidentity.Identity{ID: accountID2}},
			},
			responseSetErrs: []error{nil, nil},
		},
		{
			name: "no frozen account",

			responseAccounts: []*account.Account{},
		},
		{
			name: "a failing account does not stop the next one",

			responseAccounts: []*account.Account{
				{Identity: commonidentity.Identity{ID: accountID1}},
				{Identity: commonidentity.Identity{ID: accountID2}},
			},
			responseSetErrs: []error{errCause, nil},
		},
		{
			name: "list fails and no account is touched",

			responseListErr: errCause,

			expectErrIs: errCause,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &accountHandler{
				db: mockDB,
			}
			ctx := context.Background()

			// only frozen accounts are listed, and there is no deleted filter.
			expectFilters := map[account.Field]any{
				account.FieldCustomerID: customerID,
				account.FieldStatus:     account.StatusFrozen,
			}
			mockDB.EXPECT().AccountList(ctx, uint64(1000), "", expectFilters).Return(tt.responseAccounts, tt.responseListErr)

			for i, a := range tt.responseAccounts {
				mockDB.EXPECT().AccountSetStatus(ctx, a.ID, account.StatusActive).Return(tt.responseSetErrs[i]).Times(1)
			}

			err := h.EventCUCustomerRecovered(ctx, &cucustomer.Customer{ID: customerID})
			if tt.expectErrIs == nil {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.expectErrIs) {
				t.Errorf("Wrong match. expect: error caused by %v, got: %v", tt.expectErrIs, err)
			}
		})
	}
}
