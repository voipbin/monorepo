package dbhandler

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

const (
	paddleSelectSQL = "SELECT balance_token, balance_credit FROM billing_accounts WHERE id = ? FOR UPDATE"

	errTextDuplicateMySQL  = "Error 1062 (23000): Duplicate entry 'synthetic' for key 'ux_billing_billings_idempotency_key'"
	errTextDuplicateSQLite = "UNIQUE constraint failed: billing_billings.idempotency_key"
)

// billingInsertColumns is the column order of the billing INSERT statement.
// The query builder sorts the columns alphabetically, so the arguments of the
// statement arrive in this order.
var billingInsertColumns = []string{
	"account_id", "amount_credit", "amount_token", "balance_credit_snapshot",
	"balance_token_snapshot", "billable_units", "cost_type", "customer_id", "id",
	"idempotency_key", "rate_credit_per_unit", "rate_token_per_unit", "reference_id",
	"reference_type", "status", "tm_billing_end", "tm_billing_start", "tm_create",
	"tm_delete", "tm_update", "transaction_type", "usage_duration",
}

// exactArg is a positional sqlmock argument matcher for one named column.
type exactArg struct {
	t    *testing.T
	name string
	want any
}

func (a exactArg) Match(v driver.Value) bool {
	var ok bool
	switch w := a.want.(type) {
	case nil:
		ok = v == nil
	case []byte:
		g, isBytes := v.([]byte)
		ok = isBytes && bytes.Equal(g, w)
	case time.Time:
		g, isTime := v.(time.Time)
		ok = isTime && g.Equal(w)
	default:
		ok = reflect.DeepEqual(v, a.want)
	}
	if !ok {
		a.t.Logf("argument %q mismatch. expect: %#v, got: %#v", a.name, a.want, v)
	}
	return ok
}

// namedArgs builds the positional matchers for the given column order and
// the expected values. A column missing in the map is a test bug.
func namedArgs(t *testing.T, columns []string, want map[string]any) []driver.Value {
	t.Helper()

	if len(columns) != len(want) {
		t.Fatalf("expected values (%d) do not match the columns (%d)", len(want), len(columns))
	}

	res := make([]driver.Value, 0, len(columns))
	for _, c := range columns {
		v, ok := want[c]
		if !ok {
			t.Fatalf("no expected value for column %q", c)
		}
		res = append(res, exactArg{t: t, name: c, want: v})
	}
	return res
}

func positionalArgs(t *testing.T, values ...any) []driver.Value {
	t.Helper()

	res := make([]driver.Value, 0, len(values))
	for i, v := range values {
		res = append(res, exactArg{t: t, name: fmt.Sprintf("arg%d", i), want: v})
	}
	return res
}

// paddleTxSpec describes one of the three Paddle balance functions for the
// shared transaction scenarios.
type paddleTxSpec struct {
	funcName string
	call     func(h *handler, ctx context.Context) error

	now           time.Time
	accountID     uuid.UUID
	currentToken  int64
	currentCredit int64

	updateFragment string
	updateArgs     func(t *testing.T) []driver.Value
	insertArgs     func(t *testing.T) []driver.Value
}

// newPaddleTxSpecs returns the specs of the three Paddle balance functions
// and the billing id the util mock hands out.
func newPaddleTxSpecs() (map[string]paddleTxSpec, uuid.UUID) {
	accountID := uuid.FromStringOrNil("7c1649a0-0001-4a00-8000-000000000001")
	customerID := uuid.FromStringOrNil("7c1649a0-0002-4a00-8000-000000000002")
	idempotencyKey := uuid.FromStringOrNil("7c1649a0-0003-4a00-8000-000000000003")
	billingID := uuid.FromStringOrNil("7c1649a0-0004-4a00-8000-000000000004")
	now := time.Date(2026, 3, 17, 10, 30, 45, 0, time.UTC)

	specs := map[string]paddleTxSpec{
		"AccountPaddleAddCredit": {
			funcName: "AccountPaddleAddCredit",
			call: func(h *handler, ctx context.Context) error {
				return h.AccountPaddleAddCredit(ctx, accountID, 250000, customerID, idempotencyKey)
			},
			now:            now,
			accountID:      accountID,
			currentToken:   500,
			currentCredit:  1000000,
			updateFragment: "UPDATE billing_accounts SET balance_credit = ?, tm_update = ? WHERE id = ?",
			updateArgs: func(t *testing.T) []driver.Value {
				return positionalArgs(t, int64(1250000), now, accountID.Bytes())
			},
			insertArgs: func(t *testing.T) []driver.Value {
				return namedArgs(t, billingInsertColumns, map[string]any{
					"account_id":              accountID.Bytes(),
					"amount_credit":           int64(250000),
					"amount_token":            int64(0),
					"balance_credit_snapshot": int64(1250000),
					"balance_token_snapshot":  int64(500),
					"billable_units":          int64(0),
					"cost_type":               "",
					"customer_id":             customerID.Bytes(),
					"id":                      billingID.Bytes(),
					"idempotency_key":         idempotencyKey.Bytes(),
					"rate_credit_per_unit":    int64(0),
					"rate_token_per_unit":     int64(0),
					"reference_id":            idempotencyKey.Bytes(),
					"reference_type":          string(billing.ReferenceTypePaddleCreditPurchase),
					"status":                  string(billing.StatusEnd),
					"tm_billing_end":          now,
					"tm_billing_start":        now,
					"tm_create":               now,
					"tm_delete":               nil,
					"tm_update":               nil,
					"transaction_type":        string(billing.TransactionTypeTopUp),
					"usage_duration":          int64(0),
				})
			},
		},
		"AccountPaddleSubtractCredit": {
			funcName: "AccountPaddleSubtractCredit",
			call: func(h *handler, ctx context.Context) error {
				return h.AccountPaddleSubtractCredit(ctx, accountID, 300000, customerID, idempotencyKey)
			},
			now:            now,
			accountID:      accountID,
			currentToken:   500,
			currentCredit:  100000,
			updateFragment: "UPDATE billing_accounts SET balance_credit = ?, tm_update = ? WHERE id = ?",
			updateArgs: func(t *testing.T) []driver.Value {
				// the balance is allowed to go negative
				return positionalArgs(t, int64(-200000), now, accountID.Bytes())
			},
			insertArgs: func(t *testing.T) []driver.Value {
				return namedArgs(t, billingInsertColumns, map[string]any{
					"account_id":              accountID.Bytes(),
					"amount_credit":           int64(-300000),
					"amount_token":            int64(0),
					"balance_credit_snapshot": int64(-200000),
					"balance_token_snapshot":  int64(500),
					"billable_units":          int64(0),
					"cost_type":               "",
					"customer_id":             customerID.Bytes(),
					"id":                      billingID.Bytes(),
					"idempotency_key":         idempotencyKey.Bytes(),
					"rate_credit_per_unit":    int64(0),
					"rate_token_per_unit":     int64(0),
					"reference_id":            idempotencyKey.Bytes(),
					"reference_type":          string(billing.ReferenceTypePaddleRefund),
					"status":                  string(billing.StatusEnd),
					"tm_billing_end":          now,
					"tm_billing_start":        now,
					"tm_create":               now,
					"tm_delete":               nil,
					"tm_update":               nil,
					"transaction_type":        string(billing.TransactionTypeRefund),
					"usage_duration":          int64(0),
				})
			},
		},
		"AccountPaddleTopUpTokens": {
			funcName: "AccountPaddleTopUpTokens",
			call: func(h *handler, ctx context.Context) error {
				return h.AccountPaddleTopUpTokens(ctx, accountID, customerID, 1000, "synthetic_plan", billing.TransactionTypeAdjustment, idempotencyKey)
			},
			now:            now,
			accountID:      accountID,
			currentToken:   40,
			currentCredit:  777,
			updateFragment: "UPDATE billing_accounts SET",
			updateArgs: func(t *testing.T) []driver.Value {
				return positionalArgs(t, int64(1000), now, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), now, accountID.Bytes())
			},
			insertArgs: func(t *testing.T) []driver.Value {
				return namedArgs(t, billingInsertColumns, map[string]any{
					"account_id":              accountID.Bytes(),
					"amount_credit":           int64(0),
					"amount_token":            int64(1000),
					"balance_credit_snapshot": int64(777),
					"balance_token_snapshot":  int64(1000),
					"billable_units":          int64(0),
					"cost_type":               "",
					"customer_id":             customerID.Bytes(),
					"id":                      billingID.Bytes(),
					"idempotency_key":         idempotencyKey.Bytes(),
					"rate_credit_per_unit":    int64(0),
					"rate_token_per_unit":     int64(0),
					"reference_id":            idempotencyKey.Bytes(),
					"reference_type":          string(billing.ReferenceTypePaddleSubscription),
					"status":                  string(billing.StatusEnd),
					"tm_billing_end":          now,
					"tm_billing_start":        now,
					"tm_create":               now,
					"tm_delete":               nil,
					"tm_update":               nil,
					"transaction_type":        string(billing.TransactionTypeAdjustment),
					"usage_duration":          int64(0),
				})
			},
		},
	}

	return specs, billingID
}

// Each of the three functions below runs the same scenarios: success, begin
// error, missing account, read error, update error, insert error, duplicate
// key (both driver messages) and commit error.

func Test_AccountPaddleAddCredit(t *testing.T) {
	specs, billingID := newPaddleTxSpecs()
	runPaddleTxScenarios(t, specs["AccountPaddleAddCredit"], billingID)
}

func Test_AccountPaddleSubtractCredit(t *testing.T) {
	specs, billingID := newPaddleTxSpecs()
	runPaddleTxScenarios(t, specs["AccountPaddleSubtractCredit"], billingID)
}

func Test_AccountPaddleTopUpTokens(t *testing.T) {
	specs, billingID := newPaddleTxSpecs()
	runPaddleTxScenarios(t, specs["AccountPaddleTopUpTokens"], billingID)
}

// runPaddleTxScenarios runs every scenario of one function against its own
// sqlmock database. The names of the subtests start with the function name.
func runPaddleTxScenarios(t *testing.T, spec paddleTxSpec, billingID uuid.UUID) {
	t.Helper()

	errInjected := errors.New("synthetic database failure")

	type scenario struct {
		name string

		// the stage at which the scenario stops; the mock expectations are built up to it
		expect func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T)

		timeNowCalls    int
		uuidCreateCalls int

		expectErrIs   error
		expectErrText string
	}

	selectRows := func(spec paddleTxSpec) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"balance_token", "balance_credit"}).AddRow(spec.currentToken, spec.currentCredit)
	}
	expectUntilRead := func(mock sqlmock.Sqlmock, spec paddleTxSpec) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(paddleSelectSQL)).
			WithArgs(spec.accountID.Bytes()).
			WillReturnRows(selectRows(spec))
	}
	expectUntilUpdate := func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
		expectUntilRead(mock, spec)
		mock.ExpectExec(regexp.QuoteMeta(spec.updateFragment)).
			WithArgs(spec.updateArgs(t)...).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	expectInsert := func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) *sqlmock.ExpectedExec {
		return mock.ExpectExec(regexp.QuoteMeta("INSERT INTO billing_billings")).
			WithArgs(spec.insertArgs(t)...)
	}

	scenarios := []scenario{
		{
			name: "success",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				expectUntilUpdate(mock, spec, t)
				expectInsert(mock, spec, t).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			timeNowCalls:    1,
			uuidCreateCalls: 1,
		},
		{
			name: "begin error",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				mock.ExpectBegin().WillReturnError(errInjected)
			},
			expectErrIs:   errInjected,
			expectErrText: "could not begin transaction",
		},
		{
			name: "account not found",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(paddleSelectSQL)).
					WithArgs(spec.accountID.Bytes()).
					WillReturnRows(sqlmock.NewRows([]string{"balance_token", "balance_credit"}))
				mock.ExpectRollback()
			},
			expectErrIs: ErrNotFound,
		},
		{
			name: "read error",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(paddleSelectSQL)).
					WithArgs(spec.accountID.Bytes()).
					WillReturnError(errInjected)
				mock.ExpectRollback()
			},
			expectErrIs:   errInjected,
			expectErrText: "could not read account",
		},
		{
			name: "update error",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				expectUntilRead(mock, spec)
				mock.ExpectExec(regexp.QuoteMeta(spec.updateFragment)).
					WithArgs(spec.updateArgs(t)...).
					WillReturnError(errInjected)
				mock.ExpectRollback()
			},
			timeNowCalls:  1,
			expectErrIs:   errInjected,
			expectErrText: "could not update",
		},
		{
			name: "insert error",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				expectUntilUpdate(mock, spec, t)
				expectInsert(mock, spec, t).WillReturnError(errInjected)
				mock.ExpectRollback()
			},
			timeNowCalls:    1,
			uuidCreateCalls: 1,
			expectErrIs:     errInjected,
			expectErrText:   "could not insert billing record",
		},
		{
			// A concurrent duplicate webhook is reported as success; the balance
			// change is rolled back and nothing is committed.
			name: "duplicate key from MySQL",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				expectUntilUpdate(mock, spec, t)
				expectInsert(mock, spec, t).WillReturnError(errors.New(errTextDuplicateMySQL))
				mock.ExpectRollback()
			},
			timeNowCalls:    1,
			uuidCreateCalls: 1,
		},
		{
			name: "duplicate key from SQLite",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				expectUntilUpdate(mock, spec, t)
				expectInsert(mock, spec, t).WillReturnError(errors.New(errTextDuplicateSQLite))
				mock.ExpectRollback()
			},
			timeNowCalls:    1,
			uuidCreateCalls: 1,
		},
		{
			// After a failed commit the transaction is already finished, so no
			// rollback reaches the driver.
			name: "commit error",
			expect: func(mock sqlmock.Sqlmock, spec paddleTxSpec, t *testing.T) {
				expectUntilUpdate(mock, spec, t)
				expectInsert(mock, spec, t).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errInjected)
			},
			timeNowCalls:    1,
			uuidCreateCalls: 1,
			expectErrIs:     errInjected,
			expectErrText:   "could not commit",
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create sqlmock: %v", err)
			}
			defer func() { _ = db.Close() }()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}
			ctx := context.Background()

			now := spec.now
			if tt.timeNowCalls > 0 {
				mockUtil.EXPECT().TimeNow().Return(&now).Times(tt.timeNowCalls)
			}
			if tt.uuidCreateCalls > 0 {
				mockUtil.EXPECT().UUIDCreate().Return(billingID).Times(tt.uuidCreateCalls)
			}

			tt.expect(mock, spec, t)

			err = spec.call(h, ctx)

			if tt.expectErrIs == nil {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			} else {
				if !errors.Is(err, tt.expectErrIs) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErrIs, err)
				}
				if tt.expectErrText != "" && (err == nil || !strings.Contains(err.Error(), spec.funcName+": "+tt.expectErrText)) {
					t.Errorf("Wrong error text. expect: %q in the message, got: %v", spec.funcName+": "+tt.expectErrText, err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func Test_AccountPaddleTopUpTokens_nextTopup(t *testing.T) {
	accountID := uuid.FromStringOrNil("7c1649a0-0011-4a00-8000-000000000011")
	customerID := uuid.FromStringOrNil("7c1649a0-0012-4a00-8000-000000000012")
	idempotencyKey := uuid.FromStringOrNil("7c1649a0-0013-4a00-8000-000000000013")
	billingID := uuid.FromStringOrNil("7c1649a0-0014-4a00-8000-000000000014")

	tests := []struct {
		name string

		now             time.Time
		txnType         billing.TransactionType
		expectNextTopup time.Time
	}{
		{
			name:            "december rolls over to the first of January of the next year",
			now:             time.Date(2026, 12, 17, 10, 30, 45, 0, time.UTC),
			txnType:         billing.TransactionTypeTopUp,
			expectNextTopup: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:            "last day of a long month goes to the first of the next month",
			now:             time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
			txnType:         billing.TransactionTypeAdjustment,
			expectNextTopup: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create sqlmock: %v", err)
			}
			defer func() { _ = db.Close() }()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}

			now := tt.now
			mockUtil.EXPECT().TimeNow().Return(&now)
			mockUtil.EXPECT().UUIDCreate().Return(billingID)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(paddleSelectSQL)).
				WithArgs(accountID.Bytes()).
				WillReturnRows(sqlmock.NewRows([]string{"balance_token", "balance_credit"}).AddRow(int64(40), int64(777)))
			mock.ExpectExec(regexp.QuoteMeta("UPDATE billing_accounts SET")).
				WithArgs(positionalArgs(t, int64(1000), now, tt.expectNextTopup, now, accountID.Bytes())...).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO billing_billings")).
				WithArgs(namedArgs(t, billingInsertColumns, map[string]any{
					"account_id":              accountID.Bytes(),
					"amount_credit":           int64(0),
					"amount_token":            int64(1000),
					"balance_credit_snapshot": int64(777),
					"balance_token_snapshot":  int64(1000),
					"billable_units":          int64(0),
					"cost_type":               "",
					"customer_id":             customerID.Bytes(),
					"id":                      billingID.Bytes(),
					"idempotency_key":         idempotencyKey.Bytes(),
					"rate_credit_per_unit":    int64(0),
					"rate_token_per_unit":     int64(0),
					"reference_id":            idempotencyKey.Bytes(),
					"reference_type":          string(billing.ReferenceTypePaddleSubscription),
					"status":                  string(billing.StatusEnd),
					"tm_billing_end":          now,
					"tm_billing_start":        now,
					"tm_create":               now,
					"tm_delete":               nil,
					"tm_update":               nil,
					"transaction_type":        string(tt.txnType),
					"usage_duration":          int64(0),
				})...).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			if err := h.AccountPaddleTopUpTokens(context.Background(), accountID, customerID, 1000, "synthetic_plan", tt.txnType, idempotencyKey); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

type paddleFixture struct {
	id         uuid.UUID
	customerID uuid.UUID
	sub        string
	tmCreate   time.Time
	deleted    bool
}

func Test_AccountGetByPaddleSubscriptionID(t *testing.T) {
	// The sqlite in-memory database is shared by all tests of the process, so
	// every id and subscription id below is unique across the package.
	listToken := "2030-01-01 00:00:00.000000"

	tmActive := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tmNewer := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	activeID := uuid.FromStringOrNil("7c1649a0-0021-4a00-8000-000000000021")
	deletedNewerID := uuid.FromStringOrNil("7c1649a0-0022-4a00-8000-000000000022")
	deletedOnlyID := uuid.FromStringOrNil("7c1649a0-0023-4a00-8000-000000000023")

	tests := []struct {
		name string

		// accounts to create; deleted ones are soft-deleted after the creation
		creates        []paddleFixture
		subscriptionID string

		expectID  uuid.UUID
		expectErr error
	}{
		{
			name: "found",
			creates: []paddleFixture{
				{activeID, uuid.FromStringOrNil("7c1649a0-0031-4a00-8000-000000000031"), "sub_t1649_found", tmActive, false},
			},
			subscriptionID: "sub_t1649_found",
			expectID:       activeID,
		},
		{
			name: "newer soft-deleted account with the same subscription id is skipped",
			creates: []paddleFixture{
				{activeID, uuid.FromStringOrNil("7c1649a0-0032-4a00-8000-000000000032"), "sub_t1649_skip", tmActive, false},
				{deletedNewerID, uuid.FromStringOrNil("7c1649a0-0033-4a00-8000-000000000033"), "sub_t1649_skip", tmNewer, true},
			},
			subscriptionID: "sub_t1649_skip",
			expectID:       activeID,
		},
		{
			name: "only a soft-deleted account has the subscription id",
			creates: []paddleFixture{
				{deletedOnlyID, uuid.FromStringOrNil("7c1649a0-0034-4a00-8000-000000000034"), "sub_t1649_gone", tmActive, true},
			},
			subscriptionID: "sub_t1649_gone",
			expectErr:      ErrNotFound,
		},
		{
			name:           "unknown subscription id",
			subscriptionID: "sub_t1649_unknown",
			expectErr:      ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{db: dbTest, cache: mockCache, utilHandler: mockUtil}
			ctx := context.Background()

			// fixture setup: the ids of one case are reused by another, so the
			// rows are removed again at the end of the case (registered first,
			// so that a failure while creating a later fixture also cleans up).
			defer func() {
				for _, c := range tt.creates {
					if _, err := dbTest.Exec("DELETE FROM billing_accounts WHERE id = ?", c.id.Bytes()); err != nil {
						t.Errorf("could not clean up the fixture account: %v", err)
					}
				}
			}()
			for _, c := range tt.creates {
				tmCreate := c.tmCreate
				mockUtil.EXPECT().TimeNow().Return(&tmCreate)
				mockCache.EXPECT().AccountSet(ctx, gomock.Any())
				if err := h.AccountCreate(ctx, &account.Account{
					Identity:             commonidentity.Identity{ID: c.id, CustomerID: c.customerID},
					PaddleSubscriptionID: c.sub,
				}); err != nil {
					t.Fatalf("could not create the fixture account: %v", err)
				}

				if c.deleted {
					tmDelete := c.tmCreate.Add(time.Hour)
					mockUtil.EXPECT().TimeNow().Return(&tmDelete)
					mockCache.EXPECT().AccountSet(ctx, gomock.Any())
					if err := h.AccountDelete(ctx, c.id); err != nil {
						t.Fatalf("could not delete the fixture account: %v", err)
					}
				}
			}
			mockUtil.EXPECT().TimeGetCurTime().Return(listToken)

			res, err := h.AccountGetByPaddleSubscriptionID(ctx, tt.subscriptionID)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.ID != tt.expectID || res.PaddleSubscriptionID != tt.subscriptionID {
				t.Errorf("Wrong match. expect: %s with %s, got: %s with %s", tt.expectID, tt.subscriptionID, res.ID, res.PaddleSubscriptionID)
			}
		})
	}
}

func Test_AccountGetByPaddleSubscriptionID_listError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{db: db, utilHandler: mockUtil}

	errInjected := errors.New("synthetic database failure")
	mockUtil.EXPECT().TimeGetCurTime().Return("2030-01-01 00:00:00.000000")
	mock.ExpectQuery(regexp.QuoteMeta("FROM billing_accounts")).WillReturnError(errInjected)

	res, err := h.AccountGetByPaddleSubscriptionID(context.Background(), "sub_t1649_error")
	if !errors.Is(err, errInjected) {
		t.Errorf("Wrong match. expect: %v, got: %v", errInjected, err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("a query failure must not be reported as not found: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_isDuplicateKeyError(t *testing.T) {
	tests := []struct {
		name string
		err  error

		expectRes bool
	}{
		{"nil", nil, false},
		{"MySQL duplicate entry", errors.New(errTextDuplicateMySQL), true},
		{"SQLite unique constraint", errors.New(errTextDuplicateSQLite), true},
		{"wrapped duplicate entry", fmt.Errorf("could not execute. err: %w", errors.New(errTextDuplicateMySQL)), true},
		{"unrelated error", errors.New("connection refused"), false},
		{"other constraint error", errors.New("FOREIGN KEY constraint failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := isDuplicateKeyError(tt.err); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingCreate_duplicateIdempotencyKey(t *testing.T) {
	// The ids and the idempotency key are unique across the package, because
	// the sqlite in-memory database is shared by all tests of the process.
	firstID := uuid.FromStringOrNil("7c1649a0-0041-4a00-8000-000000000041")
	secondID := uuid.FromStringOrNil("7c1649a0-0042-4a00-8000-000000000042")
	idempotencyKey := uuid.FromStringOrNil("7c1649a0-0043-4a00-8000-000000000043")
	referenceID := uuid.FromStringOrNil("7c1649a0-0044-4a00-8000-000000000044")
	secondReferenceID := uuid.FromStringOrNil("7c1649a0-0045-4a00-8000-000000000045")

	tmCreate := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	tmBillingStart := time.Date(2024, 5, 6, 7, 0, 0, 0, time.UTC)
	tmBillingEnd := time.Date(2024, 5, 6, 7, 1, 0, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &handler{db: dbTest, cache: mockCache, utilHandler: mockUtil}
	ctx := context.Background()

	defer func() {
		if _, err := dbTest.Exec("DELETE FROM billing_billings WHERE idempotency_key = ?", idempotencyKey.Bytes()); err != nil {
			t.Errorf("could not clean up the fixture billing: %v", err)
		}
	}()

	first := &billing.Billing{
		Identity:        commonidentity.Identity{ID: firstID},
		IdempotencyKey:  idempotencyKey,
		TransactionType: billing.TransactionTypeUsage,
		Status:          billing.StatusEnd,
		ReferenceType:   billing.ReferenceTypeCall,
		ReferenceID:     referenceID,
		AmountCredit:    -111,
		TMBillingStart:  &tmBillingStart,
		TMBillingEnd:    &tmBillingEnd,
	}
	expectFirst := &billing.Billing{
		Identity:        commonidentity.Identity{ID: firstID},
		IdempotencyKey:  idempotencyKey,
		TransactionType: billing.TransactionTypeUsage,
		Status:          billing.StatusEnd,
		ReferenceType:   billing.ReferenceTypeCall,
		ReferenceID:     referenceID,
		AmountCredit:    -111,
		TMBillingStart:  &tmBillingStart,
		TMBillingEnd:    &tmBillingEnd,
		TMCreate:        &tmCreate,
	}

	// first call: stored, then the stored row is read back and cached
	var cached *billing.Billing
	mockUtil.EXPECT().TimeNow().Return(&tmCreate)
	mockCache.EXPECT().BillingSet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, b *billing.Billing) error {
		cached = b
		return nil
	})
	if err := h.BillingCreate(ctx, first); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectFirst, cached) {
		t.Errorf("Wrong cached billing.\nexpect: %v\ngot: %v", expectFirst, cached)
	}

	// second call: different id and amount, same idempotency key. It is
	// reported as success, nothing is cached and the stored row is unchanged.
	tmCreate2 := tmCreate.Add(time.Hour)
	mockUtil.EXPECT().TimeNow().Return(&tmCreate2)
	second := &billing.Billing{
		Identity:        commonidentity.Identity{ID: secondID},
		IdempotencyKey:  idempotencyKey,
		TransactionType: billing.TransactionTypeUsage,
		Status:          billing.StatusEnd,
		ReferenceType:   billing.ReferenceTypeCall,
		ReferenceID:     secondReferenceID,
		AmountCredit:    -999,
	}
	if err := h.BillingCreate(ctx, second); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.BillingGetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectFirst, res) {
		t.Errorf("the stored billing was changed by the duplicate call.\nexpect: %v\ngot: %v", expectFirst, res)
	}

	var count int
	if err := dbTest.QueryRow("SELECT COUNT(*) FROM billing_billings WHERE idempotency_key = ?", idempotencyKey.Bytes()).Scan(&count); err != nil {
		t.Fatalf("could not count the rows: %v", err)
	}
	if count != 1 {
		t.Errorf("Wrong match. expect: 1 row, got: %d", count)
	}
}
