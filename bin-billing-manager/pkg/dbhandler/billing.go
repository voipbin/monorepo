package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"monorepo/bin-billing-manager/models/billing"
)

const (
	// select query for call get
	billingSelect = `
	select
		id,
		customer_id,
		account_id,

		status,

		reference_type,
		reference_id,

		cost_per_unit,
		cost_total,

		billing_unit_count,

		tm_billing_start,
		tm_billing_end,

		tm_create,
		tm_update,
		tm_delete

	from
		billing_billings
	`
)

// billingGetFromRow gets the billing from the row.
func (h *handler) billingGetFromRow(row *sql.Rows) (*billing.Billing, error) {

	res := &billing.Billing{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.AccountID,

		&res.Status,

		&res.ReferenceType,
		&res.ReferenceID,

		&res.CostPerUnit,
		&res.CostTotal,

		&res.BillingUnitCount,

		&res.TMBillingStart,
		&res.TMBillingEnd,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. billingGetFromRow. err: %v", err)
	}

	return res, nil
}

// BillingCreate creates new billing record.
func (h *handler) BillingCreate(ctx context.Context, c *billing.Billing) error {
	q := `insert into billing_billings(
		id,
		customer_id,
		account_id,

		status,

		reference_type,
		reference_id,

		cost_per_unit,
		cost_total,

		billing_unit_count,

		tm_billing_start,
		tm_billing_end,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?,
		?, ?,
		?, ?,
		?,
		?, ?,
		?, ?, ?
	)`

	_, err := h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),
		c.AccountID.Bytes(),

		c.Status,

		c.ReferenceType,
		c.ReferenceID.Bytes(),

		c.CostPerUnit,
		c.CostTotal,

		c.BillingUnitCount,

		c.TMBillingStart,
		c.TMBillingEnd,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. BillingCreate. err: %v", err)
	}

	// update the cache
	_ = h.billingUpdateToCache(ctx, c.ID)

	return nil
}

// billingGetFromCache returns billing from the cache.
func (h *handler) billingGetFromCache(ctx context.Context, id uuid.UUID) (*billing.Billing, error) {

	// get from cache
	res, err := h.cache.BillingGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// billingGetByReferenceIDFromCache returns billing of the given reference id from the cache.
func (h *handler) billingGetByReferenceIDFromCache(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error) {

	// get from cache
	res, err := h.cache.BillingGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// billingGetFromDB returns billing from the DB.
func (h *handler) billingGetFromDB(ctx context.Context, id uuid.UUID) (*billing.Billing, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", billingSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. billingGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.billingGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get billing. billingGetFromDB, err: %v", err)
	}

	return res, nil
}

// billingGetByReferenceIDFromDB returns billing of the given reference id from the DB.
func (h *handler) billingGetByReferenceIDFromDB(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error) {

	// prepare
	q := fmt.Sprintf("%s where reference_id = ?", billingSelect)

	row, err := h.db.Query(q, referenceID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. billingGetByReferenceIDFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.billingGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get billing. billingGetByReferenceIDFromDB, err: %v", err)
	}

	return res, nil
}

// billingUpdateToCache gets the billing from the DB and update the cache.
func (h *handler) billingUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.billingGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.billingSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// billingSetToCache sets the given billing to the cache
func (h *handler) billingSetToCache(ctx context.Context, c *billing.Billing) error {
	if err := h.cache.BillingSet(ctx, c); err != nil {
		return err
	}

	return nil
}

// BillingGet returns billing.
func (h *handler) BillingGet(ctx context.Context, id uuid.UUID) (*billing.Billing, error) {

	res, err := h.billingGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.billingGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.billingSetToCache(ctx, res)

	return res, nil
}

// BillingGetByReferenceID returns billing by reference id.
func (h *handler) BillingGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error) {

	res, err := h.billingGetByReferenceIDFromCache(ctx, referenceID)
	if err == nil {
		return res, nil
	}

	res, err = h.billingGetByReferenceIDFromDB(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.billingSetToCache(ctx, res)

	return res, nil
}

// BillingGets returns a list of billing.
func (h *handler) BillingGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*billing.Billing, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, billingSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "account_id", "reference_id":
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
		return nil, fmt.Errorf("could not query. BillingGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*billing.Billing{}
	for rows.Next() {
		u, err := h.billingGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. BillingGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// BillingSetStatusEnd sets the billing status to end
func (h *handler) BillingSetStatusEnd(ctx context.Context, id uuid.UUID, billingDuration float32, timestamp string) error {
	// prepare
	q := `
	update
		billing_billings
	set
		status = ?,
		cost_total = cost_per_unit * ?,
		billing_unit_count = ?,
		tm_billing_end = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, billing.StatusEnd, billingDuration, billingDuration, timestamp, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. BillingSetStatusEnd. err: %v", err)
	}

	// update the cache
	_ = h.billingUpdateToCache(ctx, id)

	return nil
}

// BillingSetStatusEnd sets the billing status
func (h *handler) BillingSetStatus(ctx context.Context, id uuid.UUID, status billing.Status) error {
	// prepare
	q := `
	update
		billing_billings
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, status, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. BillingSetStatus. err: %v", err)
	}

	// update the cache
	_ = h.billingUpdateToCache(ctx, id)

	return nil
}

// BillingDelete deletes the billing
func (h *handler) BillingDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		billing_billings
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. BillingDelete. err: %v", err)
	}

	// update the cache
	_ = h.billingUpdateToCache(ctx, id)

	return nil
}
