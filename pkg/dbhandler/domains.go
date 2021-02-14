package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
)

const (
	domainSelect = `
	select
		id,
		user_id,

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
func (h *handler) domainGetFromRow(row *sql.Rows) (*models.Domain, error) {
	res := &models.Domain{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,

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

// DomainGetFromDB returns Domain from the DB.
func (h *handler) DomainGetFromDB(ctx context.Context, id uuid.UUID) (*models.Domain, error) {

	q := fmt.Sprintf("%s where id = ?", domainSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. DomainGetFromDB. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.domainGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. DomainGetFromDB. err: %v", err)
	}

	return res, nil
}

// DomainUpdateToCache gets the AstAuth from the DB and update the cache.
func (h *handler) DomainUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.DomainGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.DomainSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// DomainSetToCache sets the given Domain to the cache
func (h *handler) DomainSetToCache(ctx context.Context, e *models.Domain) error {
	if err := h.cache.DomainSet(ctx, e); err != nil {
		return err
	}

	return nil
}

// DomainGetFromCache returns Domain from the cache.
func (h *handler) DomainGetFromCache(ctx context.Context, id uuid.UUID) (*models.Domain, error) {

	// get from cache
	res, err := h.cache.DomainGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DomainCreate creates new Domain record.
func (h *handler) DomainCreate(ctx context.Context, b *models.Domain) error {
	q := `insert into domains(
		id,
		user_id,

		name,
		detail,
		domain_name,

		tm_create
	) values(
		?, ?,
		?, ?, ?,
		?
	)
	`

	_, err := h.db.Exec(q,
		b.ID.Bytes(),
		b.UserID,

		b.Name,
		b.Detail,
		b.DomainName,

		getCurTime(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. DomainCreate. err: %v", err)
	}

	// update the cache
	h.DomainUpdateToCache(ctx, b.ID)

	return nil
}

// DomainGet returns Domain.
func (h *handler) DomainGet(ctx context.Context, id uuid.UUID) (*models.Domain, error) {

	res, err := h.DomainGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.DomainGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.DomainSetToCache(ctx, res)

	return res, nil
}

// DomainGetByDomainName gets the domain by the given domain_name.
func (h *handler) DomainGetByDomainName(ctx context.Context, domainName string) (*models.Domain, error) {

	q := fmt.Sprintf("%s where domain_name = ? and tm_delete is NULL", domainSelect)

	row, err := h.db.Query(q, domainName)
	if err != nil {
		return nil, fmt.Errorf("could not query. DomainGetByDomainName. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.domainGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. DomainGetByDomainName. err: %v", err)
	}

	return res, nil
}

// DomainGetsByUserID returns list of domains.
func (h *handler) DomainGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*models.Domain, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete is null
			and user_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, domainSelect)

	rows, err := h.db.Query(q, userID, token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. DomainGetsByUserID. err: %v", err)
	}
	defer rows.Close()

	var res []*models.Domain
	for rows.Next() {
		u, err := h.domainGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. DomainGetsByUserID. err: %v", err)
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

	_, err := h.db.Exec(q, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. DomainDelete. err: %v", err)
	}

	// update the cache
	h.DomainUpdateToCache(ctx, id)

	return nil
}
