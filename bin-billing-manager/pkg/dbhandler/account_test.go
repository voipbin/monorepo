package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

func Test_AccountCreate(t *testing.T) {

	type test struct {
		name string

		account *account.Account

		responseCurTime *time.Time
		expectRes       *account.Account
	}

	tmCreate1 := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	tmCreate2 := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			"have all",
			&account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
					CustomerID: uuid.FromStringOrNil("6efc4a5e-0600-11ee-9aca-57553e6045e7"),
				},
				Name:          "test name",
				Detail:        "test detail",
				BalanceCredit: 9999000,
				PaymentType:   account.PaymentTypeNone,
				PaymentMethod: account.PaymentMethodNone,
			},

			&tmCreate1,
			&account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
					CustomerID: uuid.FromStringOrNil("6efc4a5e-0600-11ee-9aca-57553e6045e7"),
				},
				Name:          "test name",
				Detail:        "test detail",
				BalanceCredit: 9999000,
				PaymentType:   account.PaymentTypeNone,
				PaymentMethod: account.PaymentMethodNone,
				TMCreate:      &tmCreate1,
				TMUpdate:      nil,
				TMDelete:      nil,
			},
		},
		{
			"empty",

			&account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				},
			},

			&tmCreate2,
			&account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				},
				TMCreate: &tmCreate2,
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.account.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(context.Background(), tt.account.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountList(t *testing.T) {

	type test struct {
		name     string
		accounts []*account.Account

		filters map[account.Field]any

		responseCurTime *time.Time
		expectRes       []*account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			accounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99a99eb2-f3d7-11ee-8c0a-f7457252a2f8"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99ceaf40-f3d7-11ee-b8bb-97bb778dce9e"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
				},
			},

			filters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
			},

			responseCurTime: &tmCreate,
			expectRes: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99a99eb2-f3d7-11ee-8c0a-f7457252a2f8"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
					TMCreate: &tmCreate,
					TMUpdate: nil,
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("99ceaf40-f3d7-11ee-b8bb-97bb778dce9e"),
						CustomerID: uuid.FromStringOrNil("995d6060-f3d7-11ee-a179-2fd11cdd97a2"),
					},
					TMCreate: &tmCreate,
					TMUpdate: nil,
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
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.accounts {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().AccountSet(ctx, gomock.Any())
				_ = h.AccountCreate(ctx, c)
			}

			res, err := h.AccountList(ctx, uint64(1000), utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_AccountListByCustomerID(t *testing.T) {

	type test struct {
		name     string
		accounts []*account.Account

		customerID uuid.UUID
		size       uint64

		responseCurTime *time.Time
		expectRes       []*account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			accounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1d6fcb5a-06ca-11ee-96c1-bb6797183957"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("21e3a2a6-06ca-11ee-a265-73b6edfdaf51"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
				},
			},

			customerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
			size:       10,

			responseCurTime: &tmCreate,
			expectRes: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("21e3a2a6-06ca-11ee-a265-73b6edfdaf51"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
					TMCreate: &tmCreate,
					TMUpdate: nil,
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1d6fcb5a-06ca-11ee-96c1-bb6797183957"),
						CustomerID: uuid.FromStringOrNil("53154680-0e5a-11ee-b558-fffd4cf00337"),
					},
					TMCreate: &tmCreate,
					TMUpdate: nil,
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
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.accounts {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().AccountSet(ctx, gomock.Any())
				_ = h.AccountCreate(ctx, c)
			}

			res, err := h.AccountListByCustomerID(ctx, tt.customerID, tt.size, utilhandler.TimeGetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) < len(tt.accounts) {
				t.Errorf("Wrong match. expect: bigger than %d, got: %d", len(tt.accounts), len(res))
			}
		})
	}
}

func Test_AccountUpdate(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		id     uuid.UUID
		fields map[account.Field]any

		responseCurTime *time.Time
		expectRes       *account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("697786c6-06cc-11ee-88f6-a79092bb719c"),
				},
			},

			id: uuid.FromStringOrNil("697786c6-06cc-11ee-88f6-a79092bb719c"),
			fields: map[account.Field]any{
				account.FieldName:   "test name",
				account.FieldDetail: "test detail",
			},

			responseCurTime: &tmCreate,
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("697786c6-06cc-11ee-88f6-a79092bb719c"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: &tmCreate,
				TMUpdate: &tmCreate,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// AccountAddBalance, AccountSubtractBalance, and AccountSubtractBalanceWithCheck now use
// accountAdjustCreditWithLedger which requires SELECT ... FOR UPDATE (not supported by SQLite).
// See Test_accountAdjustCreditWithLedger_* below for sqlmock-based transaction tests.

func Test_AccountUpdatePaymentInfo(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		id     uuid.UUID
		fields map[account.Field]any

		responseCurTime *time.Time
		expectRes       *account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5d0dd0e2-06cd-11ee-a292-3bc4c472124e"),
				},
			},

			id: uuid.FromStringOrNil("5d0dd0e2-06cd-11ee-a292-3bc4c472124e"),
			fields: map[account.Field]any{
				account.FieldPaymentType:   account.PaymentTypePrepaid,
				account.FieldPaymentMethod: account.PaymentMethodCreditCard,
			},

			responseCurTime: &tmCreate,
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5d0dd0e2-06cd-11ee-a292-3bc4c472124e"),
				},
				PaymentType:   account.PaymentTypePrepaid,
				PaymentMethod: account.PaymentMethodCreditCard,
				TMCreate:      &tmCreate,
				TMUpdate:      &tmCreate,
				TMDelete:      nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AccountDelete(t *testing.T) {

	type test struct {
		name    string
		account *account.Account

		id uuid.UUID

		responseCurTime *time.Time
		expectRes       *account.Account
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),
				},
			},

			id: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),

			responseCurTime: &tmCreate,
			expectRes: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d9d1d2c-06d1-11ee-a149-033de1ce53d7"),
				},
				TMCreate: &tmCreate,
				TMUpdate: &tmCreate,
				TMDelete: &tmCreate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountCreate(ctx, tt.account); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			if err := h.AccountDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AccountGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AccountSet(ctx, gomock.Any())
			res, err := h.AccountGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// sqlmock-based tests for accountAdjustCreditWithLedger.
// These verify the full transaction flow (BEGIN → SELECT FOR UPDATE → UPDATE → INSERT → COMMIT)
// without requiring a real MySQL database.
//
// Note: After commit, accountUpdateToCache issues a SELECT query against h.db whose return value
// is discarded in production code (_ = h.accountUpdateToCache(...)). We intentionally do not mock
// this post-commit query; sqlmock returns an error for the unexpected call, which is silently ignored.

func Test_accountAdjustCreditWithLedger_add_balance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-000000000000")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 500
	var currentCredit int64 = 1000000
	var signedAmount int64 = 500000

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, signedAmount, false)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_accountAdjustCreditWithLedger_subtract_balance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-000000000001")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 500
	var currentCredit int64 = 1000000
	var signedAmount int64 = -300000

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, signedAmount, false)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_accountAdjustCreditWithLedger_subtract_with_check_sufficient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-000000000002")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 500
	var currentCredit int64 = 1000000
	var signedAmount int64 = -500000

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, signedAmount, true)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_accountAdjustCreditWithLedger_subtract_with_check_insufficient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	var currentToken int64 = 500
	var currentCredit int64 = 1000000
	var signedAmount int64 = -2000000

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectRollback()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, signedAmount, true)
	if err != ErrInsufficientBalance {
		t.Errorf("expected ErrInsufficientBalance, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_accountAdjustCreditWithLedger_account_not_found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, 500000, false)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: BeginTx fails for credit adjustment.
func Test_accountAdjustCreditWithLedger_begin_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	mock.ExpectBegin().WillReturnError(fmt.Errorf("connection refused"))

	err = h.accountAdjustCreditWithLedger(ctx, accountID, 500000, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: SELECT returns a non-ErrNoRows error for credit adjustment.
func Test_accountAdjustCreditWithLedger_select_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnError(fmt.Errorf("connection reset"))
	mock.ExpectRollback()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, 500000, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: UPDATE balance fails for credit adjustment.
func Test_accountAdjustCreditWithLedger_update_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 500
	var currentCredit int64 = 1000000

	mockUtil.EXPECT().TimeNow().Return(&now)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnError(fmt.Errorf("deadlock detected"))
	mock.ExpectRollback()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, 500000, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: INSERT billing ledger fails for credit adjustment.
func Test_accountAdjustCreditWithLedger_insert_billing_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-666666666666")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 500
	var currentCredit int64 = 1000000

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnError(fmt.Errorf("duplicate key"))
	mock.ExpectRollback()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, 500000, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: COMMIT fails for credit adjustment.
func Test_accountAdjustCreditWithLedger_commit_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-777777777777")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 500
	var currentCredit int64 = 1000000

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("commit timeout"))

	err = h.accountAdjustCreditWithLedger(ctx, accountID, 500000, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: balance exactly equals subtract amount with check (credit).
func Test_accountAdjustCreditWithLedger_subtract_exact_balance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-888888888888")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 500
	var currentCredit int64 = 1000000
	var signedAmount int64 = -1000000 // exactly matches currentCredit

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = h.accountAdjustCreditWithLedger(ctx, accountID, signedAmount, true)
	if err != nil {
		t.Errorf("expected no error (exact balance should pass check), got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// sqlmock-based tests for accountAdjustTokenWithLedger.
// These mirror the credit adjustment tests above but verify token balance operations.

func Test_accountAdjustTokenWithLedger_add_tokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-111111111111")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 1000
	var currentCredit int64 = 5000000
	var signedAmount int64 = 500

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, signedAmount, false)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_accountAdjustTokenWithLedger_subtract_tokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-111111111112")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 1000
	var currentCredit int64 = 5000000
	var signedAmount int64 = -300

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, signedAmount, false)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_accountAdjustTokenWithLedger_subtract_with_check_sufficient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-222222222222")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 1000
	var currentCredit int64 = 5000000
	var signedAmount int64 = -500

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, signedAmount, true)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_accountAdjustTokenWithLedger_subtract_with_check_insufficient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	var currentToken int64 = 1000
	var currentCredit int64 = 5000000
	var signedAmount int64 = -2000

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectRollback()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, signedAmount, true)
	if err != ErrInsufficientBalance {
		t.Errorf("expected ErrInsufficientBalance, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_accountAdjustTokenWithLedger_account_not_found(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, 500, false)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: SELECT returns a non-ErrNoRows error (e.g. connection error).
func Test_accountAdjustTokenWithLedger_select_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnError(fmt.Errorf("connection reset"))
	mock.ExpectRollback()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, 500, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: UPDATE balance fails after successful SELECT.
func Test_accountAdjustTokenWithLedger_update_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 1000
	var currentCredit int64 = 5000000

	mockUtil.EXPECT().TimeNow().Return(&now)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnError(fmt.Errorf("deadlock detected"))
	mock.ExpectRollback()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, 500, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: INSERT billing ledger fails after successful UPDATE.
func Test_accountAdjustTokenWithLedger_insert_billing_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-333333333333")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 1000
	var currentCredit int64 = 5000000

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnError(fmt.Errorf("duplicate key"))
	mock.ExpectRollback()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, 500, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: COMMIT fails after all operations succeed.
func Test_accountAdjustTokenWithLedger_commit_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-444444444444")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 1000
	var currentCredit int64 = 5000000

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("commit timeout"))

	err = h.accountAdjustTokenWithLedger(ctx, accountID, 500, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: balance exactly equals subtract amount with check (boundary — should succeed).
func Test_accountAdjustTokenWithLedger_subtract_exact_balance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-555555555555")
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	var currentToken int64 = 500
	var currentCredit int64 = 5000000
	var signedAmount int64 = -500 // exactly matches currentToken

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(billingID)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
			AddRow(customerID.Bytes(), currentToken, currentCredit))
	mock.ExpectExec("UPDATE billing_accounts SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO billing_billings").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = h.accountAdjustTokenWithLedger(ctx, accountID, signedAmount, true)
	if err != nil {
		t.Errorf("expected no error (exact balance should pass check), got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Edge case: BeginTx fails.
func Test_accountAdjustTokenWithLedger_begin_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	mock.ExpectBegin().WillReturnError(fmt.Errorf("connection refused"))

	err = h.accountAdjustTokenWithLedger(ctx, accountID, 500, false)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
