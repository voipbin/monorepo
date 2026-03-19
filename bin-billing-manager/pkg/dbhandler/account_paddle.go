package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
)

// AccountGetByPaddleSubscriptionID returns the account matching the given Paddle subscription ID.
func (h *handler) AccountGetByPaddleSubscriptionID(ctx context.Context, paddleSubscriptionID string) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "AccountGetByPaddleSubscriptionID",
		"paddle_subscription_id": paddleSubscriptionID,
	})

	// Use the same filter pattern as AccountListByCustomerID for soft-delete
	filters := map[account.Field]any{
		account.FieldPaddleSubscriptionID: paddleSubscriptionID,
		account.FieldDeleted:              false,
	}

	accounts, err := h.AccountList(ctx, 1, "", filters)
	if err != nil {
		log.Errorf("Could not list accounts by paddle subscription ID: %v", err)
		return nil, fmt.Errorf("could not query account: %w", err)
	}

	if len(accounts) == 0 {
		return nil, ErrNotFound
	}

	log.WithField("account", accounts[0]).Debugf("Retrieved account by paddle_subscription_id. account_id: %s", accounts[0].ID)
	return accounts[0], nil
}

// AccountPaddleAddCredit atomically adds credit balance and creates a Paddle billing record.
// One Paddle event → one billing record (no double-ledger).
func (h *handler) AccountPaddleAddCredit(ctx context.Context, accountID uuid.UUID, amountMicros int64, customerID uuid.UUID, idempotencyKey uuid.UUID) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("AccountPaddleAddCredit: could not begin transaction. err: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Raw SQL with FOR UPDATE: squirrel does not support SELECT...FOR UPDATE locking.
	// The pessimistic lock prevents concurrent balance modifications on the same account.
	var currentToken, currentCredit int64
	row := tx.QueryRowContext(ctx,
		"SELECT balance_token, balance_credit FROM billing_accounts WHERE id = ? FOR UPDATE",
		accountID.Bytes())
	if err := row.Scan(&currentToken, &currentCredit); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("AccountPaddleAddCredit: could not read account. err: %v", err)
	}

	now := h.utilHandler.TimeNow()
	newBalance := currentCredit + amountMicros

	// Raw SQL UPDATE within the same TX that holds the FOR UPDATE lock.
	_, err = tx.ExecContext(ctx,
		"UPDATE billing_accounts SET balance_credit = ?, tm_update = ? WHERE id = ?",
		newBalance, now, accountID.Bytes())
	if err != nil {
		return fmt.Errorf("AccountPaddleAddCredit: could not update balance. err: %v", err)
	}

	bill := &billing.Billing{}
	bill.ID = h.utilHandler.UUIDCreate()
	bill.CustomerID = customerID
	bill.AccountID = accountID
	bill.TransactionType = billing.TransactionTypeTopUp
	bill.Status = billing.StatusEnd
	bill.ReferenceType = billing.ReferenceTypePaddleCreditPurchase
	bill.ReferenceID = idempotencyKey
	bill.CostType = billing.CostTypeNone
	bill.AmountCredit = amountMicros
	bill.AmountToken = 0
	bill.BalanceCreditSnapshot = newBalance
	bill.BalanceTokenSnapshot = currentToken
	bill.IdempotencyKey = idempotencyKey
	bill.TMBillingStart = now
	bill.TMBillingEnd = now
	bill.TMCreate = now

	fields, err := commondatabasehandler.PrepareFields(bill)
	if err != nil {
		return fmt.Errorf("AccountPaddleAddCredit: could not prepare billing fields. err: %v", err)
	}
	query, args, err := sq.Insert(billingsTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AccountPaddleAddCredit: could not build insert query. err: %v", err)
	}
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("AccountPaddleAddCredit: could not insert billing record. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("AccountPaddleAddCredit: could not commit. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, accountID)
	return nil
}

// AccountPaddleSubtractCredit atomically subtracts credit balance and creates a Paddle refund billing record.
// Allows balance to go negative (caller should check and freeze if needed).
func (h *handler) AccountPaddleSubtractCredit(ctx context.Context, accountID uuid.UUID, amountMicros int64, customerID uuid.UUID, idempotencyKey uuid.UUID) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("AccountPaddleSubtractCredit: could not begin transaction. err: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Raw SQL with FOR UPDATE: squirrel does not support SELECT...FOR UPDATE locking.
	// The pessimistic lock prevents concurrent balance modifications on the same account.
	var currentToken, currentCredit int64
	row := tx.QueryRowContext(ctx,
		"SELECT balance_token, balance_credit FROM billing_accounts WHERE id = ? FOR UPDATE",
		accountID.Bytes())
	if err := row.Scan(&currentToken, &currentCredit); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("AccountPaddleSubtractCredit: could not read account. err: %v", err)
	}

	now := h.utilHandler.TimeNow()
	newBalance := currentCredit - amountMicros

	// Raw SQL UPDATE within the same TX that holds the FOR UPDATE lock.
	_, err = tx.ExecContext(ctx,
		"UPDATE billing_accounts SET balance_credit = ?, tm_update = ? WHERE id = ?",
		newBalance, now, accountID.Bytes())
	if err != nil {
		return fmt.Errorf("AccountPaddleSubtractCredit: could not update balance. err: %v", err)
	}

	bill := &billing.Billing{}
	bill.ID = h.utilHandler.UUIDCreate()
	bill.CustomerID = customerID
	bill.AccountID = accountID
	bill.TransactionType = billing.TransactionTypeRefund
	bill.Status = billing.StatusEnd
	bill.ReferenceType = billing.ReferenceTypePaddleRefund
	bill.ReferenceID = idempotencyKey
	bill.CostType = billing.CostTypeNone
	bill.AmountCredit = -amountMicros
	bill.AmountToken = 0
	bill.BalanceCreditSnapshot = newBalance
	bill.BalanceTokenSnapshot = currentToken
	bill.IdempotencyKey = idempotencyKey
	bill.TMBillingStart = now
	bill.TMBillingEnd = now
	bill.TMCreate = now

	fields, err := commondatabasehandler.PrepareFields(bill)
	if err != nil {
		return fmt.Errorf("AccountPaddleSubtractCredit: could not prepare billing fields. err: %v", err)
	}
	query, args, err := sq.Insert(billingsTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AccountPaddleSubtractCredit: could not build insert query. err: %v", err)
	}
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("AccountPaddleSubtractCredit: could not insert billing record. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("AccountPaddleSubtractCredit: could not commit. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, accountID)
	return nil
}

// AccountPaddleTopUpTokens atomically resets tokens and creates a Paddle subscription billing record.
// txnType should be TransactionTypeTopUp for new subs/renewals or TransactionTypeAdjustment for plan changes.
func (h *handler) AccountPaddleTopUpTokens(ctx context.Context, accountID uuid.UUID, customerID uuid.UUID, tokenAmount int64, planType string, txnType billing.TransactionType, idempotencyKey uuid.UUID) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("AccountPaddleTopUpTokens: could not begin transaction. err: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Raw SQL with FOR UPDATE: squirrel does not support SELECT...FOR UPDATE locking.
	// The pessimistic lock prevents concurrent balance modifications on the same account.
	var currentToken, currentCredit int64
	row := tx.QueryRowContext(ctx,
		"SELECT balance_token, balance_credit FROM billing_accounts WHERE id = ? FOR UPDATE",
		accountID.Bytes())
	if err := row.Scan(&currentToken, &currentCredit); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("AccountPaddleTopUpTokens: could not read account. err: %v", err)
	}

	now := h.utilHandler.TimeNow()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	// Raw SQL UPDATE within the same TX that holds the FOR UPDATE lock.
	// Multi-column update with computed nextMonth value makes this simpler as raw SQL.
	_, err = tx.ExecContext(ctx,
		`UPDATE billing_accounts SET
			balance_token = ?,
			tm_last_topup = ?,
			tm_next_topup = ?,
			tm_update = ?
		WHERE id = ?`,
		tokenAmount, now, nextMonth, now, accountID.Bytes())
	if err != nil {
		return fmt.Errorf("AccountPaddleTopUpTokens: could not update account. err: %v", err)
	}

	bill := &billing.Billing{}
	bill.ID = h.utilHandler.UUIDCreate()
	bill.CustomerID = customerID
	bill.AccountID = accountID
	bill.TransactionType = txnType
	bill.Status = billing.StatusEnd
	bill.ReferenceType = billing.ReferenceTypePaddleSubscription
	bill.ReferenceID = idempotencyKey
	bill.CostType = billing.CostTypeNone
	bill.AmountCredit = 0
	bill.AmountToken = tokenAmount
	bill.BalanceCreditSnapshot = currentCredit
	bill.BalanceTokenSnapshot = tokenAmount
	bill.IdempotencyKey = idempotencyKey
	bill.TMBillingStart = now
	bill.TMBillingEnd = now
	bill.TMCreate = now

	fields, err := commondatabasehandler.PrepareFields(bill)
	if err != nil {
		return fmt.Errorf("AccountPaddleTopUpTokens: could not prepare billing fields. err: %v", err)
	}
	query, args, err := sq.Insert(billingsTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AccountPaddleTopUpTokens: could not build insert query. err: %v", err)
	}
	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("AccountPaddleTopUpTokens: could not insert billing record. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("AccountPaddleTopUpTokens: could not commit. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, accountID)
	return nil
}
