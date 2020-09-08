package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
)

// UserUpdateToCache gets the user from the DB and update the cache.
func (h *handler) UserUpdateToCache(ctx context.Context, id uint64) error {

	res, err := h.UserGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.UserSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// UserSetToCache sets the given user to the cache
func (h *handler) UserSetToCache(ctx context.Context, u *user.User) error {
	if err := h.cache.UserSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// UserGetFromCache returns user from the cache.
func (h *handler) UserGetFromCache(ctx context.Context, id uint64) (*user.User, error) {

	// get from cache
	res, err := h.cache.UserGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// UserGetFromDB returns bridge from the DB.
func (h *handler) UserGetFromDB(ctx context.Context, id uint64) (*user.User, error) {

	// prepare
	q := `
	select
		id,
		username,
		password_hash,

		permission,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		users
	where
		id = ?
	`

	row, err := h.db.Query(q, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. UserGetFromDB. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.userGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. UserGetFromDB. err: %v", err)
	}

	return res, nil
}

// userGetFromRow gets the user from the row.
func (h *handler) userGetFromRow(row *sql.Rows) (*user.User, error) {
	res := &user.User{}
	if err := row.Scan(
		&res.ID,
		&res.Username,
		&res.PasswordHash,

		&res.Permission,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. userGetFromRow. err: %v", err)
	}

	return res, nil
}

// UserGet returns user.
func (h *handler) UserGet(ctx context.Context, id uint64) (*user.User, error) {
	res, err := h.UserGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.UserGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.UserSetToCache(ctx, res)

	return res, nil
}

// UserGet returns user.
func (h *handler) UserGets(ctx context.Context) ([]*user.User, error) {
	// prepare
	q := `
	select
		id,
		username,
		password_hash,

		permission,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		users
	`

	rows, err := h.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("could not query. UserGetFromDB. err: %v", err)
	}
	defer rows.Close()

	var res []*user.User
	for rows.Next() {
		u, err := h.userGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. UserGetFromDB. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// UserCreate creates new user record and returns the created user record.
func (h *handler) UserCreate(ctx context.Context, b *user.User) error {
	q := `insert into users(
		id,
		username,
		password_hash,

		permission,

		tm_create
	) values(
		?, ?, ?,
		?,
		?
		)
	`

	_, err := h.db.Exec(q,
		b.ID,
		b.Username,
		b.PasswordHash,

		b.Permission,

		b.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute. UserCreate. err: %v", err)
	}

	// update the cache
	h.UserUpdateToCache(ctx, b.ID)

	return nil
}

// UserGetByUsername returns user.
func (h *handler) UserGetByUsername(ctx context.Context, username string) (*user.User, error) {
	// prepare
	q := `
	select
		id,
		username,
		password_hash,

		permission,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		users
	where
		username = ?
	`

	row, err := h.db.Query(q, username)
	if err != nil {
		return nil, fmt.Errorf("could not query. UserGetByUsername. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.userGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. UserGetByUsername. err: %v", err)
	}

	return res, nil
}
