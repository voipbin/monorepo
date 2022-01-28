package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
)

const (
	astEndpointSelect = `
	select
		id,
		transport,
		aors,
		auth,
		context,
		identify_by,
		from_domain
	from
		ps_endpoints
	`
)

func (h *handler) astEndpointGetFromRow(row *sql.Rows) (*astendpoint.AstEndpoint, error) {
	res := &astendpoint.AstEndpoint{}
	if err := row.Scan(
		&res.ID,
		&res.Transport,
		&res.AORs,
		&res.Auth,
		&res.Context,
		&res.IdentifyBy,
		&res.FromDomain,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. bridgeGetFromRow. err: %v", err)
	}

	return res, nil
}

// AstEndpointGetFromDB returns AstEndpoint from the DB.
func (h *handler) AstEndpointGetFromDB(ctx context.Context, id string) (*astendpoint.AstEndpoint, error) {

	q := fmt.Sprintf("%s where id = ?", astEndpointSelect)

	row, err := h.db.Query(q, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. AstEndpointGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.astEndpointGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. AstEndpointGetFromDB. err: %v", err)
	}

	return res, nil
}

// AstEndpointUpdateToCache gets the AstEdnpoint from the DB and update the cache.
func (h *handler) AstEndpointUpdateToCache(ctx context.Context, id string) error {

	res, err := h.AstEndpointGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.AstEndpointSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// AstEdnpointSetToCache sets the given AstEndpoint to the cache
func (h *handler) AstEndpointSetToCache(ctx context.Context, ednpoint *astendpoint.AstEndpoint) error {
	if err := h.cache.AstEndpointSet(ctx, ednpoint); err != nil {
		return err
	}

	return nil
}

// AstEndpointGetFromCache returns AstEndpoint from the cache.
func (h *handler) AstEndpointGetFromCache(ctx context.Context, id string) (*astendpoint.AstEndpoint, error) {

	// get from cache
	res, err := h.cache.AstEndpointGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AstEndpointCreate creates new asterisk-endpoint record.
func (h *handler) AstEndpointCreate(ctx context.Context, b *astendpoint.AstEndpoint) error {
	q := `insert into ps_endpoints(
		id,
		transport,
		aors,
		auth,
		context,

		identify_by,
		from_domain
	) values(
		?, ?, ?, ?, ?,
		?, ?
		)
	`

	_, err := h.db.Exec(q,
		b.ID,
		b.Transport,
		b.AORs,
		b.Auth,
		b.Context,

		b.IdentifyBy,
		b.FromDomain,
	)
	if err != nil {
		return fmt.Errorf("could not execute. AstEndpointCreate. err: %v", err)
	}

	// update the cache
	h.AstEndpointUpdateToCache(ctx, *b.ID)

	return nil
}

// AstEndpointGet returns AstEndpoint.
func (h *handler) AstEndpointGet(ctx context.Context, id string) (*astendpoint.AstEndpoint, error) {

	res, err := h.AstEndpointGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.AstEndpointGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.AstEndpointSetToCache(ctx, res)

	return res, nil
}

// AstEndpointDelete deletes given AstEndpoint
func (h *handler) AstEndpointDelete(ctx context.Context, id string) error {

	// delete
	q := `
	delete from ps_endpoints
	where
		id = ?
	`

	_, err := h.db.Exec(q, id)
	if err != nil {
		return fmt.Errorf("could not execute. AstEndpointDelete. err: %v", err)
	}

	// delete from the cache
	h.cache.AstEndpointDel(ctx, id)

	return nil
}
