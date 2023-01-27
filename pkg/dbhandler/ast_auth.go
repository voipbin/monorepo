package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
)

const (
	astAuthSelect = `
	select
		id,
		auth_type,

		username,
		password,
		realm,

		md5_cred,
		nonce_lifetime,

		oauth_clientid,
		oauth_secret,

		refresh_token
	from
		ps_auths
	`
)

// astAuthGetFromRow gets the AstAuth from the row
func (h *handler) astAuthGetFromRow(row *sql.Rows) (*astauth.AstAuth, error) {
	res := &astauth.AstAuth{}
	if err := row.Scan(
		&res.ID,
		&res.AuthType,

		&res.Username,
		&res.Password,
		&res.Realm,

		&res.MD5Cred,
		&res.NonceLifetime,

		&res.OAuthClientID,
		&res.OAuthSecret,

		&res.RefreshToken,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. astAuthGetFromRow. err: %v", err)
	}

	return res, nil
}

// AstAuthCreate creates new asterisk-auth record.
func (h *handler) AstAuthCreate(ctx context.Context, b *astauth.AstAuth) error {
	q := `insert into ps_auths(
		id,
		auth_type,

		username,
		password,
		realm,

		md5_cred,
		nonce_lifetime,

		oauth_clientid,
		oauth_secret,

		refresh_token
	) values(
		?, ?,
		?, ?, ?,
		?, ?,
		?, ?,
		?
		)
	`

	_, err := h.db.Exec(q,
		b.ID,
		b.AuthType,

		b.Username,
		b.Password,
		b.Realm,

		b.MD5Cred,
		b.NonceLifetime,

		b.OAuthClientID,
		b.OAuthSecret,

		b.RefreshToken,
	)
	if err != nil {
		return fmt.Errorf("could not execute. AstAuthCreate. err: %v", err)
	}

	// update the cache
	_ = h.astAuthUpdateToCache(ctx, *b.ID)

	return nil
}

// astAuthGetFromDB returns AstAuth from the DB.
func (h *handler) astAuthGetFromDB(ctx context.Context, id string) (*astauth.AstAuth, error) {

	q := fmt.Sprintf("%s where id = ?", astAuthSelect)

	row, err := h.db.Query(q, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. AstAuthGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.astAuthGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. AstAuthGetFromDB. err: %v", err)
	}

	return res, nil
}

// astAuthUpdateToCache gets the AstAuth from the DB and update the cache.
func (h *handler) astAuthUpdateToCache(ctx context.Context, id string) error {

	res, err := h.astAuthGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.astAuthSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// astAuthSetToCache sets the given AstAuth to the cache
func (h *handler) astAuthSetToCache(ctx context.Context, auth *astauth.AstAuth) error {
	if err := h.cache.AstAuthSet(ctx, auth); err != nil {
		return err
	}

	return nil
}

// astAuthGetFromCache returns AstAuth from the cache.
func (h *handler) astAuthGetFromCache(ctx context.Context, id string) (*astauth.AstAuth, error) {

	// get from cache
	res, err := h.cache.AstAuthGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// astAuthDeleteFromCache returns AstAuth from the cache.
func (h *handler) astAuthDeleteFromCache(ctx context.Context, id string) error {
	// delete from the cache
	return h.cache.AstAuthDel(ctx, id)
}

// AstAuthGet returns AstAuth.
func (h *handler) AstAuthGet(ctx context.Context, id string) (*astauth.AstAuth, error) {

	res, err := h.astAuthGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.astAuthGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.astAuthSetToCache(ctx, res)

	return res, nil
}

// AstAuthDelete deletes given AstAuth
func (h *handler) AstAuthDelete(ctx context.Context, id string) error {

	// delete
	q := `
	delete from ps_auths
	where
		id = ?
	`

	_, err := h.db.Exec(q, id)
	if err != nil {
		return fmt.Errorf("could not execute. AstAuthCreate. err: %v", err)
	}

	// delete from the cache
	_ = h.astAuthDeleteFromCache(ctx, id)

	return nil
}

// AstAuthDelete deletes given AstAuth
func (h *handler) AstAuthUpdate(ctx context.Context, auth *astauth.AstAuth) error {

	// query
	q := `
	update ps_auths set
		password = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, auth.Password, auth.ID)
	if err != nil {
		return fmt.Errorf("could not execute. AstAuthUpdate. err: %v", err)
	}

	// update to the cache
	_ = h.astAuthUpdateToCache(ctx, *auth.ID)

	return nil
}
