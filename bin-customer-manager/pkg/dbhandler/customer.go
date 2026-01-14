package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-customer-manager/models/customer"
)

const (
	customerTable = "customer_customers"
)

// customerGetFromRow gets the customer from the row.
func (h *handler) customerGetFromRow(row *sql.Rows) (*customer.Customer, error) {
	res := &customer.Customer{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. customerGetFromRow. err: %v", err)
	}

	return res, nil
}

// CustomerCreate creates new customer record and returns the created customer record.
func (h *handler) CustomerCreate(ctx context.Context, c *customer.Customer) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = DefaultTimeStamp
	c.TMDelete = DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CustomerCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(customerTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CustomerCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. CustomerCreate. err: %v", err)
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
	fields := commondatabasehandler.GetDBFields(&customer.Customer{})
	query, args, err := squirrel.
		Select(fields...).
		From(customerTable).
		Where(squirrel.Eq{string(customer.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. customerGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. customerGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. customerGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.customerGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. customerGetFromDB. id: %s, err: %v", id, err)
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
func (h *handler) CustomerGets(ctx context.Context, size uint64, token string, filters map[customer.Field]any) ([]*customer.Customer, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&customer.Customer{})
	sb := squirrel.
		Select(fields...).
		From(customerTable).
		Where(squirrel.Lt{string(customer.FieldTMCreate): token}).
		OrderBy(string(customer.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. CustomerGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CustomerGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CustomerGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*customer.Customer{}
	for rows.Next() {
		u, err := h.customerGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. CustomerGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. CustomerGets. err: %v", err)
	}

	return res, nil
}

// CustomerUpdate updates customer fields.
func (h *handler) CustomerUpdate(ctx context.Context, id uuid.UUID, fields map[customer.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[customer.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CustomerUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(customerTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(customer.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("CustomerUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("CustomerUpdate: exec failed: %w", err)
	}

	_ = h.customerUpdateToCache(ctx, id)
	return nil
}

// CustomerDelete deletes the customer.
func (h *handler) CustomerDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[customer.Field]any{
		customer.FieldTMUpdate: ts,
		customer.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CustomerDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(customerTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(customer.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("CustomerDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("CustomerDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	// update the cache
	_ = h.customerUpdateToCache(ctx, id)

	return nil
}
