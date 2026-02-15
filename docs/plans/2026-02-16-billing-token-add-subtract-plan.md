# Billing Token Add/Subtract Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add token add/subtract operations to billing-control CLI and dbhandler, mirroring the existing credit adjustment pattern with ledger tracking.

**Architecture:** New `accountAdjustTokenWithLedger` function atomically adjusts `balance_token` and inserts a billing ledger entry within a single transaction. Public wrappers expose add/subtract/subtract-with-check. The accounthandler layer delegates to dbhandler and returns the updated account. CLI commands wire into the accounthandler.

**Tech Stack:** Go, MySQL (via go-sqlmock for tests), squirrel query builder, gomock, cobra CLI

---

### Task 1: Add ReferenceTypeTokenAdjustment constant

**Files:**
- Modify: `bin-billing-manager/models/billing/billing.go:77` (after `ReferenceTypeCreditAdjustment`)

**Step 1: Add the constant**

In `models/billing/billing.go`, add `ReferenceTypeTokenAdjustment` after `ReferenceTypeCreditAdjustment` (line 77):

```go
	ReferenceTypeCreditAdjustment  ReferenceType = "credit_adjustment"
	ReferenceTypeTokenAdjustment   ReferenceType = "token_adjustment"
```

**Step 2: Verify it compiles**

Run: `cd bin-billing-manager && go build ./...`
Expected: BUILD SUCCESS

**Step 3: Commit**

```bash
git add bin-billing-manager/models/billing/billing.go
git commit -m "NOJIRA-Billing-token-add-subtract

- bin-billing-manager: Add ReferenceTypeTokenAdjustment constant"
```

---

### Task 2: Add accountAdjustTokenWithLedger and public wrappers to dbhandler

**Files:**
- Modify: `bin-billing-manager/pkg/dbhandler/account.go:323` (after `AccountSubtractBalance`)
- Modify: `bin-billing-manager/pkg/dbhandler/main.go:31` (after `AccountTopUpTokens` in interface)

**Step 1: Add the three new methods to DBHandler interface**

In `pkg/dbhandler/main.go`, add after line 31 (`AccountTopUpTokens`):

```go
	AccountAddTokens(ctx context.Context, accountID uuid.UUID, amount int64) error
	AccountSubtractTokens(ctx context.Context, accountID uuid.UUID, amount int64) error
	AccountSubtractTokensWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) error
```

**Step 2: Add accountAdjustTokenWithLedger and wrappers**

In `pkg/dbhandler/account.go`, add after `AccountSubtractBalance` (line 323):

```go
// accountAdjustTokenWithLedger atomically adjusts the account token balance and creates a ledger entry.
// signedAmount is positive for additions and negative for subtractions.
// If checkBalance is true, returns ErrInsufficientBalance when the current token balance is insufficient.
func (h *handler) accountAdjustTokenWithLedger(ctx context.Context, accountID uuid.UUID, signedAmount int64, checkBalance bool) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accountAdjustTokenWithLedger: could not begin transaction. err: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Lock account row and read current balances + customer_id
	var customerID []byte
	var currentToken, currentCredit int64
	row := tx.QueryRowContext(ctx,
		"SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE id = ? FOR UPDATE",
		accountID.Bytes())
	if err := row.Scan(&customerID, &currentToken, &currentCredit); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("accountAdjustTokenWithLedger: could not read account. err: %v", err)
	}

	// Balance check for subtract-with-check
	if checkBalance && signedAmount < 0 && currentToken < -signedAmount {
		return ErrInsufficientBalance
	}

	now := h.utilHandler.TimeNow()
	newBalance := currentToken + signedAmount

	// Update token balance
	_, err = tx.ExecContext(ctx,
		"UPDATE billing_accounts SET balance_token = ?, tm_update = ? WHERE id = ?",
		newBalance, now, accountID.Bytes())
	if err != nil {
		return fmt.Errorf("accountAdjustTokenWithLedger: could not update balance. err: %v", err)
	}

	// Parse customer_id from raw bytes
	parsedCustomerID, err := uuid.FromBytes(customerID)
	if err != nil {
		return fmt.Errorf("accountAdjustTokenWithLedger: could not parse customer_id. err: %v", err)
	}

	// Insert ledger entry
	ledgerEntry := &billing.Billing{}
	ledgerEntry.ID = h.utilHandler.UUIDCreate()
	ledgerEntry.CustomerID = parsedCustomerID
	ledgerEntry.AccountID = accountID
	ledgerEntry.TransactionType = billing.TransactionTypeAdjustment
	ledgerEntry.Status = billing.StatusEnd
	ledgerEntry.ReferenceType = billing.ReferenceTypeTokenAdjustment
	ledgerEntry.ReferenceID = ledgerEntry.ID
	ledgerEntry.AmountToken = signedAmount
	ledgerEntry.AmountCredit = 0
	ledgerEntry.BalanceTokenSnapshot = newBalance
	ledgerEntry.BalanceCreditSnapshot = currentCredit
	ledgerEntry.TMBillingStart = now
	ledgerEntry.TMBillingEnd = now
	ledgerEntry.TMCreate = now

	fields, err := commondatabasehandler.PrepareFields(ledgerEntry)
	if err != nil {
		return fmt.Errorf("accountAdjustTokenWithLedger: could not prepare billing fields. err: %v", err)
	}

	query, args, err := sq.Insert(billingsTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("accountAdjustTokenWithLedger: could not build billing insert query. err: %v", err)
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("accountAdjustTokenWithLedger: could not insert billing record. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accountAdjustTokenWithLedger: could not commit. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, accountID)
	return nil
}

// AccountAddTokens adds the value to the account token balance and creates a ledger entry.
func (h *handler) AccountAddTokens(ctx context.Context, accountID uuid.UUID, amount int64) error {
	return h.accountAdjustTokenWithLedger(ctx, accountID, amount, false)
}

// AccountSubtractTokensWithCheck atomically checks the token balance is sufficient and subtracts the amount.
// Returns ErrInsufficientBalance if the account token balance is less than the amount.
func (h *handler) AccountSubtractTokensWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) error {
	return h.accountAdjustTokenWithLedger(ctx, accountID, -amount, true)
}

// AccountSubtractTokens subtracts the value from the account token balance and creates a ledger entry.
func (h *handler) AccountSubtractTokens(ctx context.Context, accountID uuid.UUID, amount int64) error {
	return h.accountAdjustTokenWithLedger(ctx, accountID, -amount, false)
}
```

**Step 3: Regenerate mocks**

Run: `cd bin-billing-manager && go generate ./...`
Expected: `mock_main.go` in `pkg/dbhandler/` is updated with the 3 new methods

**Step 4: Run tests**

Run: `cd bin-billing-manager && go test ./...`
Expected: ALL PASS (existing tests still work, new mock methods are generated)

**Step 5: Commit**

```bash
git add bin-billing-manager/pkg/dbhandler/account.go bin-billing-manager/pkg/dbhandler/main.go bin-billing-manager/pkg/dbhandler/mock_main.go
git commit -m "NOJIRA-Billing-token-add-subtract

- bin-billing-manager: Add accountAdjustTokenWithLedger and public wrappers
- bin-billing-manager: Add AccountAddTokens, AccountSubtractTokens, AccountSubtractTokensWithCheck to DBHandler interface"
```

---

### Task 3: Add sqlmock tests for accountAdjustTokenWithLedger

**Files:**
- Modify: `bin-billing-manager/pkg/dbhandler/account_test.go` (append after existing credit tests at end of file)

**Step 1: Add 5 sqlmock tests**

Append to `pkg/dbhandler/account_test.go` after the last function (`Test_accountAdjustCreditWithLedger_account_not_found`):

```go
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
	now := time.Date(2024, 2, 20, 14, 0, 0, 0, time.UTC)

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
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-222222222222")
	now := time.Date(2024, 2, 20, 14, 0, 0, 0, time.UTC)

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
	billingID := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-333333333333")
	now := time.Date(2024, 2, 20, 14, 0, 0, 0, time.UTC)

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
```

**Step 2: Run tests**

Run: `cd bin-billing-manager && go test ./pkg/dbhandler/... -v -run "Test_accountAdjustTokenWithLedger"`
Expected: ALL 5 PASS

**Step 3: Commit**

```bash
git add bin-billing-manager/pkg/dbhandler/account_test.go
git commit -m "NOJIRA-Billing-token-add-subtract

- bin-billing-manager: Add sqlmock tests for accountAdjustTokenWithLedger"
```

---

### Task 4: Add AddTokens, SubtractTokens, SubtractTokensWithCheck to accounthandler

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/main.go:33` (after `UpdatePlanType` in interface)
- Modify: `bin-billing-manager/pkg/accounthandler/db.go:182` (after `AddBalance`)

**Step 1: Add to AccountHandler interface**

In `pkg/accounthandler/main.go`, add after line 32 (`UpdatePlanType`):

```go
	AddTokens(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
	SubtractTokens(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
	SubtractTokensWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error)
```

**Step 2: Add implementations**

In `pkg/accounthandler/db.go`, add after `AddBalance` (line 182):

```go
// AddTokens adds tokens to the given account.
func (h *accountHandler) AddTokens(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "AddTokens",
		"account_id": accountID,
		"amount":     amount,
	})

	if err := h.db.AccountAddTokens(ctx, accountID, amount); err != nil {
		log.Errorf("Could not add tokens. err: %v", err)
		return nil, errors.Wrap(err, "could not add tokens")
	}

	res, err := h.db.AccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account info")
	}

	return res, nil
}

// SubtractTokens subtracts tokens from the given account.
func (h *accountHandler) SubtractTokens(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SubtractTokens",
		"account_id": accountID,
		"amount":     amount,
	})

	if err := h.db.AccountSubtractTokens(ctx, accountID, amount); err != nil {
		log.Errorf("Could not subtract tokens. err: %v", err)
		return nil, errors.Wrap(err, "could not subtract tokens")
	}

	res, err := h.db.AccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account info")
	}

	return res, nil
}

// SubtractTokensWithCheck atomically checks the token balance and subtracts.
// For unlimited plan accounts, the balance check is skipped.
func (h *accountHandler) SubtractTokensWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SubtractTokensWithCheck",
		"account_id": accountID,
		"amount":     amount,
	})

	// get account to check plan type
	a, err := h.db.AccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return nil, errors.Wrap(err, "could not get account info")
	}

	// unlimited plan accounts bypass balance check
	if a.PlanType == account.PlanTypeUnlimited {
		return h.SubtractTokens(ctx, accountID, amount)
	}

	// other accounts use atomic check-and-subtract
	if err := h.db.AccountSubtractTokensWithCheck(ctx, accountID, amount); err != nil {
		log.Errorf("Could not subtract tokens with check. err: %v", err)
		return nil, errors.Wrap(err, "could not subtract tokens")
	}

	res, err := h.db.AccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get updated account. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated account info")
	}

	return res, nil
}
```

**Step 3: Regenerate mocks**

Run: `cd bin-billing-manager && go generate ./...`
Expected: `mock_main.go` in `pkg/accounthandler/` is updated with the 3 new methods

**Step 4: Run tests**

Run: `cd bin-billing-manager && go test ./...`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add bin-billing-manager/pkg/accounthandler/main.go bin-billing-manager/pkg/accounthandler/db.go bin-billing-manager/pkg/accounthandler/mock_main.go
git commit -m "NOJIRA-Billing-token-add-subtract

- bin-billing-manager: Add AddTokens, SubtractTokens, SubtractTokensWithCheck to AccountHandler"
```

---

### Task 5: Add gomock tests for accounthandler token methods

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/db_test.go` (append after existing balance tests)

**Step 1: Add tests for AddTokens, SubtractTokens, SubtractTokensWithCheck**

Append to `pkg/accounthandler/db_test.go` after `Test_SubtractBalanceWithCheck_get_error`:

```go
func Test_AddTokens(t *testing.T) {
	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount *account.Account
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("aa111111-0000-0000-0000-000000000001"),
			amount:    500,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa111111-0000-0000-0000-000000000001"),
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

			mockDB.EXPECT().AccountAddTokens(ctx, tt.accountID, tt.amount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			res, err := h.AddTokens(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseAccount, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_AddTokens_error(t *testing.T) {
	tests := []struct {
		name      string
		accountID uuid.UUID
		amount    int64
	}{
		{name: "db add error", accountID: uuid.FromStringOrNil("aa222222-0000-0000-0000-000000000001"), amount: 500},
		{name: "db get error", accountID: uuid.FromStringOrNil("aa222222-0000-0000-0000-000000000002"), amount: 500},
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

			if tt.name == "db add error" {
				mockDB.EXPECT().AccountAddTokens(ctx, tt.accountID, tt.amount).Return(fmt.Errorf("add failed"))
				_, err := h.AddTokens(ctx, tt.accountID, tt.amount)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			mockDB.EXPECT().AccountAddTokens(ctx, tt.accountID, tt.amount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(nil, fmt.Errorf("get failed"))
			_, err := h.AddTokens(ctx, tt.accountID, tt.amount)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_SubtractTokens(t *testing.T) {
	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount *account.Account
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("aa333333-0000-0000-0000-000000000001"),
			amount:    300,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa333333-0000-0000-0000-000000000001"),
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

			mockDB.EXPECT().AccountSubtractTokens(ctx, tt.accountID, tt.amount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			res, err := h.SubtractTokens(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseAccount, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_SubtractTokens_error(t *testing.T) {
	tests := []struct {
		name      string
		accountID uuid.UUID
		amount    int64
	}{
		{name: "db subtract error", accountID: uuid.FromStringOrNil("aa444444-0000-0000-0000-000000000001"), amount: 300},
		{name: "db get error", accountID: uuid.FromStringOrNil("aa444444-0000-0000-0000-000000000002"), amount: 300},
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

			if tt.name == "db subtract error" {
				mockDB.EXPECT().AccountSubtractTokens(ctx, tt.accountID, tt.amount).Return(fmt.Errorf("subtract failed"))
				_, err := h.SubtractTokens(ctx, tt.accountID, tt.amount)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			mockDB.EXPECT().AccountSubtractTokens(ctx, tt.accountID, tt.amount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(nil, fmt.Errorf("get failed"))
			_, err := h.SubtractTokens(ctx, tt.accountID, tt.amount)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_SubtractTokensWithCheck_normal(t *testing.T) {
	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount        *account.Account
		responseUpdatedAccount *account.Account
	}{
		{
			name: "normal account uses atomic check",

			accountID: uuid.FromStringOrNil("aa555555-0000-0000-0000-000000000001"),
			amount:    500,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa555555-0000-0000-0000-000000000001"),
				},
				PlanType:     account.PlanTypeFree,
				BalanceToken: 1000,
			},
			responseUpdatedAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa555555-0000-0000-0000-000000000001"),
				},
				PlanType:     account.PlanTypeFree,
				BalanceToken: 500,
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

			// get account to check type
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)
			// atomic check-and-subtract
			mockDB.EXPECT().AccountSubtractTokensWithCheck(ctx, tt.accountID, tt.amount).Return(nil)
			// get updated account
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseUpdatedAccount, nil)

			res, err := h.SubtractTokensWithCheck(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseUpdatedAccount, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAccount, res)
			}
		})
	}
}

func Test_SubtractTokensWithCheck_unlimited(t *testing.T) {
	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount        *account.Account
		responseUpdatedAccount *account.Account
	}{
		{
			name: "unlimited plan account bypasses check",

			accountID: uuid.FromStringOrNil("aa666666-0000-0000-0000-000000000001"),
			amount:    5000,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa666666-0000-0000-0000-000000000001"),
				},
				PlanType:     account.PlanTypeUnlimited,
				BalanceToken: 100,
			},
			responseUpdatedAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa666666-0000-0000-0000-000000000001"),
				},
				PlanType:     account.PlanTypeUnlimited,
				BalanceToken: -4900,
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

			// get account to check plan type — unlimited
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)
			// unlimited bypasses to SubtractTokens (no check)
			mockDB.EXPECT().AccountSubtractTokens(ctx, tt.accountID, tt.amount).Return(nil)
			// get updated account
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseUpdatedAccount, nil)

			res, err := h.SubtractTokensWithCheck(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseUpdatedAccount, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAccount, res)
			}
		})
	}
}

func Test_SubtractTokensWithCheck_insufficient(t *testing.T) {
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

	accountID := uuid.FromStringOrNil("aa777777-0000-0000-0000-000000000001")

	mockDB.EXPECT().AccountGet(ctx, accountID).Return(&account.Account{
		Identity:     commonidentity.Identity{ID: accountID},
		PlanType:     account.PlanTypeFree,
		BalanceToken: 100,
	}, nil)
	mockDB.EXPECT().AccountSubtractTokensWithCheck(ctx, accountID, int64(5000)).Return(fmt.Errorf("insufficient balance"))

	_, err := h.SubtractTokensWithCheck(ctx, accountID, 5000)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_SubtractTokensWithCheck_get_error(t *testing.T) {
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

	accountID := uuid.FromStringOrNil("aa888888-0000-0000-0000-000000000001")

	mockDB.EXPECT().AccountGet(ctx, accountID).Return(nil, fmt.Errorf("initial get failed"))

	_, err := h.SubtractTokensWithCheck(ctx, accountID, int64(500))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
```

**Step 2: Run tests**

Run: `cd bin-billing-manager && go test ./pkg/accounthandler/... -v -run "Test_(Add|Subtract)Tokens"`
Expected: ALL 8 PASS

**Step 3: Commit**

```bash
git add bin-billing-manager/pkg/accounthandler/db_test.go
git commit -m "NOJIRA-Billing-token-add-subtract

- bin-billing-manager: Add gomock tests for AddTokens, SubtractTokens, SubtractTokensWithCheck"
```

---

### Task 6: Add add-tokens and subtract-tokens CLI commands

**Files:**
- Modify: `bin-billing-manager/cmd/billing-control/main.go:71` (command registration) and after `runAccountSubtractBalance` (line 441)

**Step 1: Register new subcommands**

In `cmd/billing-control/main.go`, add after `cmdAccountSubtractBalance()` registration (line 71):

```go
	cmdAccount.AddCommand(cmdAccountAddTokens())
	cmdAccount.AddCommand(cmdAccountSubtractTokens())
```

**Step 2: Add command definitions and run functions**

Add after `runAccountSubtractBalance` (line 441):

```go
func cmdAccountAddTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-tokens",
		Short: "Add tokens to an account",
		RunE:  runAccountAddTokens,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")
	flags.Int64("amount", 0, "Token amount to add (required)")

	return cmd
}

func cmdAccountSubtractTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subtract-tokens",
		Short: "Subtract tokens from an account",
		RunE:  runAccountSubtractTokens,
	}

	flags := cmd.Flags()
	flags.String("id", "", "Account ID (required)")
	flags.Int64("amount", 0, "Token amount to subtract (required)")

	return cmd
}

func runAccountAddTokens(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	amount := viper.GetInt64("amount")
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	res, err := accountHandler.AddTokens(context.Background(), targetID, amount)
	if err != nil {
		return errors.Wrap(err, "failed to add tokens")
	}

	return printJSON(res)
}

func runAccountSubtractTokens(cmd *cobra.Command, args []string) error {
	accountHandler, _, err := initHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to initialize handlers")
	}

	targetID, err := resolveUUID("id", "Account ID")
	if err != nil {
		return errors.Wrap(err, "invalid account ID format")
	}

	amount := viper.GetInt64("amount")
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	res, err := accountHandler.SubtractTokens(context.Background(), targetID, amount)
	if err != nil {
		return errors.Wrap(err, "failed to subtract tokens")
	}

	return printJSON(res)
}
```

**Step 3: Verify it builds**

Run: `cd bin-billing-manager && go build ./cmd/billing-control/...`
Expected: BUILD SUCCESS

**Step 4: Commit**

```bash
git add bin-billing-manager/cmd/billing-control/main.go
git commit -m "NOJIRA-Billing-token-add-subtract

- bin-billing-manager: Add add-tokens and subtract-tokens CLI commands to billing-control"
```

---

### Task 7: Full verification and final commit

**Step 1: Run full verification workflow**

```bash
cd bin-billing-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS, no lint errors

**Step 2: If any issues, fix them before proceeding**

**Step 3: Commit any vendor/mod changes if needed**

```bash
git add -A
git status
# Only commit if there are changes (vendor updates, go.sum changes, etc.)
git commit -m "NOJIRA-Billing-token-add-subtract

- bin-billing-manager: Run go mod tidy and vendor sync"
```

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-Billing-token-add-subtract
```

Then create a PR with:
- Title: `NOJIRA-Billing-token-add-subtract`
- Body: Summary of all changes with project prefixes
