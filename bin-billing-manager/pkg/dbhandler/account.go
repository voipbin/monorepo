package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
)

const (
	accountsTable = "billing_accounts"
)

// accountGetFromRow gets the account from the row.
func (h *handler) accountGetFromRow(row *sql.Rows) (*account.Account, error) {
	res := &account.Account{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. accountGetFromRow. err: %v", err)
	}

	return res, nil
}

// AccountCreate creates new account record.
func (h *handler) AccountCreate(ctx context.Context, c *account.Account) error {
	c.TMCreate = h.utilHandler.TimeNow()
	c.TMUpdate = nil
	c.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("AccountCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(accountsTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AccountCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AccountCreate: could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, c.ID)

	return nil
}

// accountGetFromCache returns account from the cache.
func (h *handler) accountGetFromCache(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	res, err := h.cache.AccountGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// accountGetFromDB returns account from the DB.
func (h *handler) accountGetFromDB(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	cols := commondatabasehandler.GetDBFields(account.Account{})

	query, args, err := sq.Select(cols...).
		From(accountsTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("accountGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("accountGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.accountGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("accountGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// accountUpdateToCache gets the account from the DB and update the cache.
func (h *handler) accountUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.accountGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.accountSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// accountSetToCache sets the given account to the cache
func (h *handler) accountSetToCache(ctx context.Context, c *account.Account) error {
	if err := h.cache.AccountSet(ctx, c); err != nil {
		return err
	}

	return nil
}

// AccountGet returns account.
func (h *handler) AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	res, err := h.accountGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.accountGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.accountSetToCache(ctx, res)

	return res, nil
}

// AccountList returns a list of account.
func (h *handler) AccountList(ctx context.Context, size uint64, token string, filters map[account.Field]any) ([]*account.Account, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(account.Account{})

	builder := sq.Select(cols...).
		From(accountsTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("AccountList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AccountList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AccountList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*account.Account{}
	for rows.Next() {
		u, err := h.accountGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("AccountList: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}

// AccountListByCustomerID returns a list of account.
func (h *handler) AccountListByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*account.Account, error) {
	filters := map[account.Field]any{
		account.FieldCustomerID: customerID,
		account.FieldDeleted:    false,
	}

	return h.AccountList(ctx, size, token, filters)
}

// AccountUpdate updates the account fields.
func (h *handler) AccountUpdate(ctx context.Context, id uuid.UUID, fields map[account.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("AccountUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(accountsTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AccountUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AccountUpdate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// accountAdjustCreditWithLedger atomically adjusts the account credit balance and creates a ledger entry.
// signedAmount is positive for additions and negative for subtractions.
// If checkBalance is true, returns ErrInsufficientBalance when the current balance is insufficient.
func (h *handler) accountAdjustCreditWithLedger(ctx context.Context, accountID uuid.UUID, signedAmount int64, checkBalance bool) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accountAdjustCreditWithLedger: could not begin transaction. err: %v", err)
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
		return fmt.Errorf("accountAdjustCreditWithLedger: could not read account. err: %v", err)
	}

	// Balance check for subtract-with-check
	if checkBalance && signedAmount < 0 && currentCredit < -signedAmount {
		return ErrInsufficientBalance
	}

	now := h.utilHandler.TimeNow()
	newBalance := currentCredit + signedAmount

	// Update balance
	_, err = tx.ExecContext(ctx,
		"UPDATE billing_accounts SET balance_credit = ?, tm_update = ? WHERE id = ?",
		newBalance, now, accountID.Bytes())
	if err != nil {
		return fmt.Errorf("accountAdjustCreditWithLedger: could not update balance. err: %v", err)
	}

	// Parse customer_id from raw bytes
	parsedCustomerID, err := uuid.FromBytes(customerID)
	if err != nil {
		return fmt.Errorf("accountAdjustCreditWithLedger: could not parse customer_id. err: %v", err)
	}

	// Insert ledger entry
	ledgerEntry := &billing.Billing{}
	ledgerEntry.ID = h.utilHandler.UUIDCreate()
	ledgerEntry.CustomerID = parsedCustomerID
	ledgerEntry.AccountID = accountID
	ledgerEntry.TransactionType = billing.TransactionTypeAdjustment
	ledgerEntry.Status = billing.StatusEnd
	ledgerEntry.ReferenceType = billing.ReferenceTypeCreditAdjustment
	ledgerEntry.AmountToken = 0
	ledgerEntry.AmountCredit = signedAmount
	ledgerEntry.BalanceTokenSnapshot = currentToken
	ledgerEntry.BalanceCreditSnapshot = newBalance
	ledgerEntry.TMBillingStart = now
	ledgerEntry.TMBillingEnd = now
	ledgerEntry.TMCreate = now

	fields, err := commondatabasehandler.PrepareFields(ledgerEntry)
	if err != nil {
		return fmt.Errorf("accountAdjustCreditWithLedger: could not prepare billing fields. err: %v", err)
	}

	query, args, err := sq.Insert(billingsTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("accountAdjustCreditWithLedger: could not build billing insert query. err: %v", err)
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("accountAdjustCreditWithLedger: could not insert billing record. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accountAdjustCreditWithLedger: could not commit. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, accountID)
	return nil
}

// AccountAddBalance adds the value to the account balance and creates a ledger entry.
func (h *handler) AccountAddBalance(ctx context.Context, accountID uuid.UUID, amount int64) error {
	return h.accountAdjustCreditWithLedger(ctx, accountID, amount, false)
}

// AccountSubtractBalanceWithCheck atomically checks the balance is sufficient and subtracts the amount.
// Returns ErrInsufficientBalance if the account balance is less than the amount.
func (h *handler) AccountSubtractBalanceWithCheck(ctx context.Context, accountID uuid.UUID, amount int64) error {
	return h.accountAdjustCreditWithLedger(ctx, accountID, -amount, true)
}

// AccountSubtractBalance subtracts the value from the account balance and creates a ledger entry.
func (h *handler) AccountSubtractBalance(ctx context.Context, accountID uuid.UUID, amount int64) error {
	return h.accountAdjustCreditWithLedger(ctx, accountID, -amount, false)
}

// AccountDelete deletes the account
func (h *handler) AccountDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(accountsTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AccountDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AccountDelete: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}
