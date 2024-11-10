package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
)

const (
	// select query for call get
	customerSelect = `
	select
		id,

		name,
		detail,

		email,
		phone_number,
		address,

		webhook_method,
		webhook_uri,

		billing_account_id,

		tm_create,
		tm_update,
		tm_delete
	from
		customer_customers
	`
)

// customerGetFromRow gets the customer from the row.
func (h *handler) customerGetFromRow(row *sql.Rows) (*customer.Customer, error) {
	res := &customer.Customer{}
	if err := row.Scan(
		&res.ID,

		&res.Name,
		&res.Detail,

		&res.Email,
		&res.PhoneNumber,
		&res.Address,

		&res.WebhookMethod,
		&res.WebhookURI,

		&res.BillingAccountID,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. customerGetFromRow. err: %v", err)
	}

	return res, nil
}

// CustomerCreate creates new customer record and returns the created customer record.
func (h *handler) CustomerCreate(ctx context.Context, c *customer.Customer) error {
	q := `insert into customer_customers(
		id,

		name,
		detail,

		email,
		phone_number,
		address,

		webhook_method,
		webhook_uri,


		billing_account_id,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?,
		?, ?,
		?, ?, ?,
		?, ?,
		?,
		?, ?, ?
		)
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q,
		c.ID.Bytes(),

		c.Name,
		c.Detail,

		c.Email,
		c.PhoneNumber,
		c.Address,

		c.WebhookMethod,
		c.WebhookURI,

		c.BillingAccountID.Bytes(),

		ts,
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. CustomerCreate. err: %v", err)
	}

	// update the cache
	_ = h.customerUpdateToCache(ctx, c.ID)

	return nil
}

// customerUpdateToCache gets the customer from the DB and update the cache.
func (h *handler) customerUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.customerGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.customerSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// customerSetToCache sets the given customer to the cache
func (h *handler) customerSetToCache(ctx context.Context, u *customer.Customer) error {
	if err := h.cache.CustomerSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// customerGetFromCache returns customer from the cache.
func (h *handler) customerGetFromCache(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {

	// get from cache
	res, err := h.cache.CustomerGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// customerGetFromDB returns customer from the DB.
func (h *handler) customerGetFromDB(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", customerSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. customerGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.customerGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. customerGetFromDB. err: %v", err)
	}

	return res, nil
}

// CustomerGet returns customer.
func (h *handler) CustomerGet(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	res, err := h.customerGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.customerGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.customerSetToCache(ctx, res)

	return res, nil
}

// CustomerGets returns customers.
func (h *handler) CustomerGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*customer.Customer, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, customerSelect)

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		case "billing_account_id":
			q = fmt.Sprintf("%s and billing_account_id = ?", q)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CustomerGets. err: %v", err)
	}
	defer rows.Close()

	var res []*customer.Customer
	for rows.Next() {
		u, err := h.customerGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. CustomerGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// CustomerDelete deletes the customer.
func (h *handler) CustomerDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		customer_customers
	set
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CustomerDelete. err: %v", err)
	}

	// update the cache
	_ = h.customerUpdateToCache(ctx, id)

	return nil
}

// CustomerSetBasicInfo sets the customer's basic info.
func (h *handler) CustomerSetBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod customer.WebhookMethod,
	webhookURI string,
) error {
	// prepare
	q := `
	update
		customer_customers
	set
		name = ?,
		detail = ?,
		email = ?,
		phone_number = ?,
		address = ?,
		webhook_method = ?,
		webhook_uri = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, name, detail, email, phoneNumber, address, webhookMethod, webhookURI, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CustomerSetBasicInfo. err: %v", err)
	}

	// update the cache
	_ = h.customerUpdateToCache(ctx, id)

	return nil
}

// CustomerSetPermission sets the customer's permission.
func (h *handler) CustomerSetPermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) error {
	// prepare
	q := `
	update
		customer_customers
	set
		permission_ids = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpPermissionIDs, err := json.Marshal(permissionIDs)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(q, tmpPermissionIDs, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CustomerSetPermission. err: %v", err)
	}

	// update the cache
	_ = h.customerUpdateToCache(ctx, id)

	return nil
}

// CustomerSetPasswordHash sets the customer's password_hash.
func (h *handler) CustomerSetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	// prepare
	q := `
	update
		customer_customers
	set
		password_hash = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, passwordHash, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CustomerSetPasswordHash. err: %v", err)
	}

	// update the cache
	_ = h.customerUpdateToCache(ctx, id)

	return nil
}

// CustomerSetBillingAccountID sets the customer's billing_account_id.
func (h *handler) CustomerSetBillingAccountID(ctx context.Context, id uuid.UUID, billingAccountID uuid.UUID) error {
	// prepare
	q := `
	update
		customer_customers
	set
		billing_account_id = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, billingAccountID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CustomerSetBillingAccountID. err: %v", err)
	}

	// update the cache
	_ = h.customerUpdateToCache(ctx, id)

	return nil
}
