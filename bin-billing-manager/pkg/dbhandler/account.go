package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
)

const (
	// select query for call get
	accountSelect = `
	select
		id,
		customer_id,

		type,

		name,
		detail,

		balance,

		payment_type,
		payment_method,

		tm_create,
		tm_update,
		tm_delete

	from
		billing_accounts
	`
)

// accountGetFromRow gets the account from the row.
func (h *handler) accountGetFromRow(row *sql.Rows) (*account.Account, error) {

	res := &account.Account{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Type,

		&res.Name,
		&res.Detail,

		&res.Balance,

		&res.PaymentType,
		&res.PaymentMethod,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. accountGetFromRow. err: %v", err)
	}

	return res, nil
}

// AccountCreate creates new account record.
func (h *handler) AccountCreate(ctx context.Context, c *account.Account) error {
	q := `insert into billing_accounts(
		id,
		customer_id,

		type,

		name,
		detail,

		balance,

		payment_type,
		payment_method,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?,
		?, ?,
		?,
		?, ?,
		?, ?, ?
	)`

	_, err := h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),

		c.Type,

		c.Name,
		c.Detail,

		c.Balance,

		c.PaymentType,
		c.PaymentMethod,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. AccountCreate. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, c.ID)

	return nil
}

// accountGetFromCache returns account from the cache.
func (h *handler) accountGetFromCache(ctx context.Context, id uuid.UUID) (*account.Account, error) {

	// get from cache
	res, err := h.cache.AccountGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// accountGetFromDB returns account from the DB.
func (h *handler) accountGetFromDB(ctx context.Context, id uuid.UUID) (*account.Account, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", accountSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. accountGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.accountGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get account. accountGetFromDB, err: %v", err)
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

// AccountGets returns a list of account.
func (h *handler) AccountGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*account.Account, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, accountSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AccountGets. err: %v", err)
	}
	defer rows.Close()

	res := []*account.Account{}
	for rows.Next() {
		u, err := h.accountGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. AccountGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// AccountGetsByCustomerID returns a list of account.
func (h *handler) AccountGetsByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*account.Account, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			customer_id = ?
			and tm_create < ?
			and tm_delete >= ?
		order by
			tm_create desc
		limit ?
		`, accountSelect)

	rows, err := h.db.Query(q, customerID.Bytes(), token, DefaultTimeStamp, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. AccountGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	res := []*account.Account{}
	for rows.Next() {
		u, err := h.accountGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. AccountGetsByCustomerID, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// AccountSet sets the account
func (h *handler) AccountSet(ctx context.Context, id uuid.UUID, name string, detail string) error {
	// prepare
	q := `
	update
		billing_accounts
	set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, name, detail, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccountSet. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// AccountAddBalance add the value to the account balance
func (h *handler) AccountAddBalance(ctx context.Context, accountID uuid.UUID, balance float32) error {
	// prepare
	q := `
	update
		billing_accounts
	set
		balance = balance + ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, balance, h.utilHandler.TimeGetCurTime(), accountID.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccountAddBalance. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, accountID)

	return nil
}

// AccountSubtractBalance substract the value from the account balance
func (h *handler) AccountSubtractBalance(ctx context.Context, accountID uuid.UUID, balance float32) error {
	// prepare
	q := `
	update
		billing_accounts
	set
		balance = balance - ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, balance, h.utilHandler.TimeGetCurTime(), accountID.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccountSubtractBalance. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, accountID)

	return nil
}

// AccountSetPaymentInfo sets the account payment settings
func (h *handler) AccountSetPaymentInfo(ctx context.Context, id uuid.UUID, paymentType account.PaymentType, paymentMethod account.PaymentMethod) error {
	// prepare
	q := `
	update
		billing_accounts
	set
		payment_type = ?,
		payment_method = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, paymentType, paymentMethod, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccountSetPayments. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// AccountDelete deletes the account
func (h *handler) AccountDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		billing_accounts
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccountDelete. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}
