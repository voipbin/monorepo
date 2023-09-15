package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
)

const (
	domainSelect = `
	select
		id,
		customer_id,

		name,
		detail,
		domain_name,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		domains
	`
)

// domainGetFromRow gets the domain from the row
func (h *handler) domainGetFromRow(row *sql.Rows) (*domain.Domain, error) {
	res := &domain.Domain{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,
		&res.DomainName,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. domainGetFromRow. err: %v", err)
	}

	return res, nil
}

// DomainCreate creates new Domain record.
func (h *handler) DomainCreate(ctx context.Context, b *domain.Domain) error {
	q := `insert into domains(
		id,
		customer_id,

		name,
		detail,
		domain_name,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?, ?,
		?, ?, ?
	)
	`

	_, err := h.db.Exec(q,
		b.ID.Bytes(),
		b.CustomerID.Bytes(),

		b.Name,
		b.Detail,
		b.DomainName,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. DomainCreate. err: %v", err)
	}

	// update the cache
	_ = h.domainUpdateToCache(ctx, b.ID)

	return nil
}

// domainGetFromDB returns Domain from the DB.
func (h *handler) domainGetFromDB(ctx context.Context, id uuid.UUID) (*domain.Domain, error) {

	q := fmt.Sprintf("%s where id = ?", domainSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. DomainGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.domainGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. DomainGetFromDB. err: %v", err)
	}

	return res, nil
}

// domainUpdateToCache gets the AstAuth from the DB and update the cache.
func (h *handler) domainUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.domainGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.domainSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// domainSetToCache sets the given Domain to the cache
func (h *handler) domainSetToCache(ctx context.Context, e *domain.Domain) error {
	if err := h.cache.DomainSet(ctx, e); err != nil {
		return err
	}

	return nil
}

// domainGetFromCache returns Domain from the cache.
func (h *handler) domainGetFromCache(ctx context.Context, id uuid.UUID) (*domain.Domain, error) {

	// get from cache
	res, err := h.cache.DomainGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// domainGetByDomainNameFromCache returns Domain from the cache.
func (h *handler) domainGetByDomainNameFromCache(ctx context.Context, domainName string) (*domain.Domain, error) {

	// get from cache
	res, err := h.cache.DomainGetByDomainName(ctx, domainName)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// domainDeleteFromCache deletes Domain from the cache.
//
//nolint:unused // good to have. will use in the future
func (h *handler) domainDeleteFromCache(ctx context.Context, id uuid.UUID, name string) error {

	// get from cache
	if err := h.cache.DomainDel(ctx, id, name); err != nil {
		return err
	}

	return nil
}

// DomainUpdateBasicInfo updates new Domain record.
func (h *handler) DomainUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	q := `
	update domains set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		name,
		detail,
		h.utilHandler.TimeGetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. DomainUpdateBasicInfo. err: %v", err)
	}

	// update the cache
	_ = h.domainUpdateToCache(ctx, id)

	return nil
}

// DomainGet returns Domain.
func (h *handler) DomainGet(ctx context.Context, id uuid.UUID) (*domain.Domain, error) {

	res, err := h.domainGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.domainGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.domainSetToCache(ctx, res)

	return res, nil
}

// DomainGetByDomainName gets the domain by the given domain_name.
func (h *handler) DomainGetByDomainName(ctx context.Context, domainName string) (*domain.Domain, error) {

	res, err := h.domainGetByDomainNameFromCache(ctx, domainName)
	if err == nil {
		return res, nil
	}

	q := fmt.Sprintf(`%s
		where
			domain_name = ?
		order by
			tm_create desc
		`, domainSelect)

	row, err := h.db.Query(q, domainName)
	if err != nil {
		return nil, fmt.Errorf("could not query. DomainGetByDomainName. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err = h.domainGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. DomainGetByDomainName. err: %v", err)
	}

	// set to the cache
	_ = h.domainSetToCache(ctx, res)

	return res, nil
}

// DomainGetsByCustomerID returns list of domains.
func (h *handler) DomainGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*domain.Domain, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			customer_id = ?
			and tm_create < ?
			and tm_delete >= ?
		order by
			tm_create desc, id desc
		limit ?
	`, domainSelect)

	rows, err := h.db.Query(q, customerID.Bytes(), token, DefaultTimeStamp, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. DomainGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	res := []*domain.Domain{}
	for rows.Next() {
		u, err := h.domainGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. DomainGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// DomainDelete deletes given Domain
func (h *handler) DomainDelete(ctx context.Context, id uuid.UUID) error {

	q := `
	update domains set
		tm_delete = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. DomainDelete. err: %v", err)
	}

	_ = h.domainUpdateToCache(ctx, id)

	return nil
}
