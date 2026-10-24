package accounthandler

import (
	"context"
	"errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_UpdateBasicInfo(t *testing.T) {

	accountID := uuid.FromStringOrNil("4a5b6c7d-0001-11ef-9a48-7f8011d36229")
	errCause := errors.New("update failure")

	type test struct {
		name string

		name2  string
		detail string

		responseUpdate error
		responseGet    *account.Account
		responseGetErr error

		expectFields  map[account.Field]any
		expectGetCall bool
		expectErrIs   error
		expectNotFnd  bool
	}

	tests := []test{
		{
			name: "normal",

			name2:  "new name",
			detail: "new detail",

			responseGet: &account.Account{
				Identity: commonidentity.Identity{ID: accountID},
				Name:     "new name",
				Detail:   "new detail",
			},

			expectFields: map[account.Field]any{
				account.FieldName:   "new name",
				account.FieldDetail: "new detail",
			},
			expectGetCall: true,
		},
		{
			name: "update fails and the account is not read back",

			name2:  "new name",
			detail: "new detail",

			responseUpdate: errCause,

			expectFields: map[account.Field]any{
				account.FieldName:   "new name",
				account.FieldDetail: "new detail",
			},
			expectErrIs: errCause,
		},
		{
			name: "read back fails",

			name2:  "n",
			detail: "d",

			responseGetErr: errCause,

			expectFields: map[account.Field]any{
				account.FieldName:   "n",
				account.FieldDetail: "d",
			},
			expectGetCall: true,
			expectErrIs:   errCause,
		},
		{
			name: "read back finds no account",

			name2:  "n",
			detail: "d",

			responseGetErr: dbhandler.ErrNotFound,

			expectFields: map[account.Field]any{
				account.FieldName:   "n",
				account.FieldDetail: "d",
			},
			expectGetCall: true,
			expectErrIs:   dbhandler.ErrNotFound,
			expectNotFnd:  true,
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

			mockDB.EXPECT().AccountUpdate(ctx, accountID, tt.expectFields).Return(tt.responseUpdate)
			if tt.expectGetCall {
				mockDB.EXPECT().AccountGet(ctx, accountID).Return(tt.responseGet, tt.responseGetErr)
			}

			res, err := h.UpdateBasicInfo(ctx, accountID, tt.name2, tt.detail)
			checkUpdateResult(t, res, err, tt.responseGet, tt.expectErrIs, tt.expectNotFnd)
		})
	}
}

func Test_UpdatePaymentInfo(t *testing.T) {

	accountID := uuid.FromStringOrNil("4a5b6c7d-0002-11ef-9a48-7f8011d36229")
	errCause := errors.New("update failure")

	type test struct {
		name string

		paymentType   account.PaymentType
		paymentMethod account.PaymentMethod

		responseUpdate error
		responseGet    *account.Account
		responseGetErr error

		expectFields  map[account.Field]any
		expectGetCall bool
		expectErrIs   error
		expectNotFnd  bool
	}

	tests := []test{
		{
			name: "normal",

			paymentType:   account.PaymentTypePrepaid,
			paymentMethod: account.PaymentMethodCreditCard,

			responseGet: &account.Account{
				Identity:      commonidentity.Identity{ID: accountID},
				PaymentType:   account.PaymentTypePrepaid,
				PaymentMethod: account.PaymentMethodCreditCard,
			},

			expectFields: map[account.Field]any{
				account.FieldPaymentType:   account.PaymentTypePrepaid,
				account.FieldPaymentMethod: account.PaymentMethodCreditCard,
			},
			expectGetCall: true,
		},
		{
			name: "update fails and the account is not read back",

			paymentType:   account.PaymentTypePrepaid,
			paymentMethod: account.PaymentMethodCreditCard,

			responseUpdate: errCause,

			expectFields: map[account.Field]any{
				account.FieldPaymentType:   account.PaymentTypePrepaid,
				account.FieldPaymentMethod: account.PaymentMethodCreditCard,
			},
			expectErrIs: errCause,
		},
		{
			name: "read back fails",

			paymentType:   account.PaymentTypePrepaid,
			paymentMethod: account.PaymentMethodCreditCard,

			responseGetErr: errCause,

			expectFields: map[account.Field]any{
				account.FieldPaymentType:   account.PaymentTypePrepaid,
				account.FieldPaymentMethod: account.PaymentMethodCreditCard,
			},
			expectGetCall: true,
			expectErrIs:   errCause,
		},
		{
			name: "read back finds no account",

			paymentType:   account.PaymentTypePrepaid,
			paymentMethod: account.PaymentMethodCreditCard,

			responseGetErr: dbhandler.ErrNotFound,

			expectFields: map[account.Field]any{
				account.FieldPaymentType:   account.PaymentTypePrepaid,
				account.FieldPaymentMethod: account.PaymentMethodCreditCard,
			},
			expectGetCall: true,
			expectErrIs:   dbhandler.ErrNotFound,
			expectNotFnd:  true,
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

			mockDB.EXPECT().AccountUpdate(ctx, accountID, tt.expectFields).Return(tt.responseUpdate)
			if tt.expectGetCall {
				mockDB.EXPECT().AccountGet(ctx, accountID).Return(tt.responseGet, tt.responseGetErr)
			}

			res, err := h.UpdatePaymentInfo(ctx, accountID, tt.paymentType, tt.paymentMethod)
			checkUpdateResult(t, res, err, tt.responseGet, tt.expectErrIs, tt.expectNotFnd)
		})
	}
}

// checkUpdateResult asserts the outcome shared by the update wrappers: the very
// account that was read back on success, otherwise an error that keeps the cause
// and, for a missing account, the typed not found error.
func checkUpdateResult(t *testing.T, res *account.Account, err error, expectAccount *account.Account, expectErrIs error, expectNotFound bool) {
	t.Helper()

	if expectErrIs == nil {
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res != expectAccount {
			t.Errorf("Wrong match. expect: %v, got: %v", expectAccount, res)
		}
		return
	}

	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil account, got: %v", res)
	}
	if !errors.Is(err, expectErrIs) {
		t.Errorf("Wrong match. expect: error caused by %v, got: %v", expectErrIs, err)
	}

	var ve *cerrors.VoipbinError
	isTyped := errors.As(err, &ve)
	if isTyped != expectNotFound {
		t.Fatalf("Wrong match. expect typed error: %v, got: %v (%v)", expectNotFound, isTyped, err)
	}
	if expectNotFound && ve.Reason != "BILLING_ACCOUNT_NOT_FOUND" {
		t.Errorf("Wrong match. expect reason: BILLING_ACCOUNT_NOT_FOUND, got: %s", ve.Reason)
	}
}

func Test_SetStatus(t *testing.T) {

	accountID := uuid.FromStringOrNil("4a5b6c7d-0003-11ef-9a48-7f8011d36229")
	errCause := errors.New("db failure")

	type test struct {
		name string

		status account.Status

		expectSetCall  bool
		responseSetErr error
		expectGetCall  bool
		responseGet    *account.Account
		responseGetErr error
		expectPublish  bool

		expectErrIs  error
		expectReason string
	}

	tests := []test{
		{
			name: "frozen",

			status: account.StatusFrozen,

			expectSetCall: true,
			expectGetCall: true,
			responseGet: &account.Account{
				Identity: commonidentity.Identity{ID: accountID},
				Status:   account.StatusFrozen,
			},
			expectPublish: true,
		},
		{
			name: "active",

			status: account.StatusActive,

			expectSetCall: true,
			expectGetCall: true,
			responseGet: &account.Account{
				Identity: commonidentity.Identity{ID: accountID},
				Status:   account.StatusActive,
			},
			expectPublish: true,
		},
		{
			name: "invalid status touches nothing",

			status: account.Status("suspended"),

			expectReason: "INVALID_ACCOUNT_STATUS",
		},
		{
			name: "empty status is invalid",

			status: account.Status(""),

			expectReason: "INVALID_ACCOUNT_STATUS",
		},
		{
			name: "set status fails",

			status: account.StatusFrozen,

			expectSetCall:  true,
			responseSetErr: errCause,

			expectErrIs: errCause,
		},
		{
			name: "read back fails and nothing is published",

			status: account.StatusFrozen,

			expectSetCall:  true,
			expectGetCall:  true,
			responseGetErr: errCause,

			expectErrIs: errCause,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accountHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			if tt.expectSetCall {
				mockDB.EXPECT().AccountSetStatus(ctx, accountID, tt.status).Return(tt.responseSetErr)
			}
			if tt.expectGetCall {
				mockDB.EXPECT().AccountGet(ctx, accountID).Return(tt.responseGet, tt.responseGetErr)
			}
			if tt.expectPublish {
				mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountUpdated, tt.responseGet)
			}

			res, err := h.SetStatus(ctx, accountID, tt.status)

			if tt.expectErrIs == nil && tt.expectReason == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if res != tt.responseGet {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.responseGet, res)
				}
				return
			}

			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil account, got: %v", res)
			}
			if tt.expectErrIs != nil && !errors.Is(err, tt.expectErrIs) {
				t.Errorf("Wrong match. expect: error caused by %v, got: %v", tt.expectErrIs, err)
			}
			if tt.expectReason != "" {
				var ve *cerrors.VoipbinError
				if !errors.As(err, &ve) {
					t.Fatalf("Wrong match. expect: typed error, got: %v", err)
				}
				if ve.Reason != tt.expectReason {
					t.Errorf("Wrong match. expect reason: %s, got: %s", tt.expectReason, ve.Reason)
				}
			}
		})
	}
}

func Test_GetByPaddleSubscriptionID(t *testing.T) {

	errCause := errors.New("db failure")

	type test struct {
		name string

		subscriptionID string

		responseAccount *account.Account
		responseErr     error
	}

	tests := []test{
		{
			name: "found",

			subscriptionID: "sub_01test0000000000000000001",

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("4a5b6c7d-0004-11ef-9a48-7f8011d36229")},
			},
		},
		{
			name: "db error is passed through unchanged",

			subscriptionID: "sub_01test0000000000000000002",

			responseErr: errCause,
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

			mockDB.EXPECT().AccountGetByPaddleSubscriptionID(ctx, tt.subscriptionID).Return(tt.responseAccount, tt.responseErr)

			res, err := h.GetByPaddleSubscriptionID(ctx, tt.subscriptionID)
			if err != tt.responseErr {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseErr, err)
			}
			if res != tt.responseAccount {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseAccount, res)
			}
		})
	}
}
