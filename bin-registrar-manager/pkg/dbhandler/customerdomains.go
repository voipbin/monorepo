package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-registrar-manager/models/customerdomain"
)

const (
	customerDomainsTable = "registrar_customer_domains"
)

// mysqlErrDuplicateEntry is the MySQL error number for a unique-key violation.
const mysqlErrDuplicateEntry = 1062

// isDuplicateEntryErr reports whether the given driver error is a unique-key
// violation. Covers MySQL (production) and sqlite (unit tests).
func isDuplicateEntryErr(err error) bool {
	if err == nil {
		return false
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == mysqlErrDuplicateEntry
	}

	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "Duplicate entry")
}

// customerDomainGetFromRow gets the customer domain from the row
func (h *handler) customerDomainGetFromRow(row *sql.Rows) (*customerdomain.CustomerDomain, error) {
	res := &customerdomain.CustomerDomain{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. customerDomainGetFromRow. err: %v", err)
	}

	return res, nil
}

// customerDomainSetToCache sets the given customer domain to the cache
func (h *handler) customerDomainSetToCache(ctx context.Context, cd *customerdomain.CustomerDomain) error {
	if err := h.cache.CustomerDomainSet(ctx, cd); err != nil {
		return err
	}

	return nil
}

// customerDomainDeleteFromCacheByRealm deletes the customer domain of the given realm from the cache.
func (h *handler) customerDomainDeleteFromCacheByRealm(ctx context.Context, realm string) error {
	if err := h.cache.CustomerDomainDelByRealm(ctx, realm); err != nil {
		return err
	}

	return nil
}

// CustomerDomainCreate creates new CustomerDomain record.
// Returns an error wrapping ErrDuplicate on a unique-key violation
// (domain_label/realm collision) so the caller can retry with a new label.
func (h *handler) CustomerDomainCreate(ctx context.Context, cd *customerdomain.CustomerDomain) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	cd.TMCreate = now
	cd.TMUpdate = nil

	fields, err := commondatabasehandler.PrepareFields(cd)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CustomerDomainCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(customerDomainsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CustomerDomainCreate. err: %v", err)
	}

	if _, errExec := h.db.ExecContext(ctx, query, args...); errExec != nil {
		if isDuplicateEntryErr(errExec) {
			return fmt.Errorf("could not execute query. CustomerDomainCreate. err: %v: %w", errExec, ErrDuplicate)
		}
		return fmt.Errorf("could not execute query. CustomerDomainCreate. err: %v", errExec)
	}

	// update the cache
	_ = h.customerDomainSetToCache(ctx, cd)

	return nil
}

// customerDomainGetFromDB returns CustomerDomain from the DB.
func (h *handler) customerDomainGetFromDB(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error) {
	fields := commondatabasehandler.GetDBFields(&customerdomain.CustomerDomain{})
	query, args, err := squirrel.
		Select(fields...).
		From(customerDomainsTable).
		Where(squirrel.Eq{string(customerdomain.FieldCustomerID): customerID.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. customerDomainGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. customerDomainGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. customerDomainGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.customerDomainGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. customerDomainGetFromDB. customer_id: %s, err: %v", customerID, err)
	}

	return res, nil
}

// CustomerDomainGet returns CustomerDomain of the given customer.
func (h *handler) CustomerDomainGet(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error) {
	return h.customerDomainGetFromDB(ctx, customerID)
}

// CustomerDomainGetByRealm returns CustomerDomain of the given realm.
// The realm lookup is Redis-cached (realm -> row) — it sits on the incoming
// call hot path (call-manager realm resolution).
func (h *handler) CustomerDomainGetByRealm(ctx context.Context, realm string) (*customerdomain.CustomerDomain, error) {

	res, err := h.cache.CustomerDomainGetByRealm(ctx, realm)
	if err == nil {
		return res, nil
	}

	query, args, err := squirrel.
		Select(commondatabasehandler.GetDBFields(&customerdomain.CustomerDomain{})...).
		From(customerDomainsTable).
		Where(squirrel.Eq{string(customerdomain.FieldRealm): realm}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. CustomerDomainGetByRealm. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CustomerDomainGetByRealm. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. CustomerDomainGetByRealm. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err = h.customerDomainGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. CustomerDomainGetByRealm. realm: %s, err: %v", realm, err)
	}

	// set to the cache
	_ = h.customerDomainSetToCache(ctx, res)

	return res, nil
}

// CustomerDomainList returns customer domains.
func (h *handler) CustomerDomainList(ctx context.Context, size uint64, token string, filters map[customerdomain.Field]any) ([]*customerdomain.CustomerDomain, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&customerdomain.CustomerDomain{})
	sb := squirrel.
		Select(fields...).
		From(customerDomainsTable).
		Where(squirrel.Lt{string(customerdomain.FieldTMCreate): token}).
		OrderBy(string(customerdomain.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. CustomerDomainList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CustomerDomainList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CustomerDomainList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*customerdomain.CustomerDomain{}
	for rows.Next() {
		u, err := h.customerDomainGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. CustomerDomainList. err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. CustomerDomainList. err: %v", err)
	}

	return res, nil
}

// CustomerDomainUpdate updates customer domain record with given fields.
// The realm cache entry for the OLD realm is invalidated (the realm is the
// cache key, so an in-place cache refresh would leave the stale key behind);
// the new row is re-cached afterwards. Returns an error wrapping ErrDuplicate
// on a unique-key violation (label/realm collision) so the caller can retry.
func (h *handler) CustomerDomainUpdate(ctx context.Context, customerID uuid.UUID, fields map[customerdomain.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	// get the current row for the old realm cache invalidation
	old, err := h.customerDomainGetFromDB(ctx, customerID)
	if err != nil {
		return fmt.Errorf("CustomerDomainUpdate: could not get current record: %w", err)
	}

	fields[customerdomain.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CustomerDomainUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(customerDomainsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(customerdomain.FieldCustomerID): customerID.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("CustomerDomainUpdate: build SQL failed: %w", err)
	}

	if _, errExec := h.db.ExecContext(ctx, sqlStr, args...); errExec != nil {
		if isDuplicateEntryErr(errExec) {
			return fmt.Errorf("CustomerDomainUpdate: exec failed: %v: %w", errExec, ErrDuplicate)
		}
		return fmt.Errorf("CustomerDomainUpdate: exec failed: %w", errExec)
	}

	// invalidate the old realm cache entry and re-cache the updated row
	_ = h.customerDomainDeleteFromCacheByRealm(ctx, old.Realm)
	if updated, errGet := h.customerDomainGetFromDB(ctx, customerID); errGet == nil {
		_ = h.customerDomainSetToCache(ctx, updated)
	}

	return nil
}

// CustomerDomainDelete deletes given CustomerDomain.
// Hard delete (no tm_delete column — deliberate deviation from sibling
// registrar tables; a pure mapping row has no soft-delete consumer).
func (h *handler) CustomerDomainDelete(ctx context.Context, customerID uuid.UUID) error {
	// get the current row for the realm cache invalidation
	old, err := h.customerDomainGetFromDB(ctx, customerID)
	if err != nil {
		return fmt.Errorf("CustomerDomainDelete: could not get current record: %w", err)
	}

	sb := squirrel.Delete(customerDomainsTable).
		Where(squirrel.Eq{string(customerdomain.FieldCustomerID): customerID.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("CustomerDomainDelete: build SQL failed: %w", err)
	}

	if _, errExec := h.db.ExecContext(ctx, sqlStr, args...); errExec != nil {
		return fmt.Errorf("CustomerDomainDelete: exec failed: %w", errExec)
	}

	// invalidate the realm cache entry
	_ = h.customerDomainDeleteFromCacheByRealm(ctx, old.Realm)

	return nil
}
