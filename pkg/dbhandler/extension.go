package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
)

const (
	extensionSelect = `
	select
		id,
		customer_id,

		name,
		detail,

		endpoint_id,
		aor_id,
		auth_id,

		extension,
		domain_name,

		realm,
		username,
		password,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		extensions
	`
)

// extensionGetFromRow gets the extension from the row
func (h *handler) extensionGetFromRow(row *sql.Rows) (*extension.Extension, error) {
	res := &extension.Extension{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,

		&res.EndpointID,
		&res.AORID,
		&res.AuthID,

		&res.Extension,
		&res.DomainName,

		&res.Realm,
		&res.Username,
		&res.Password,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. extensionGetFromRow. err: %v", err)
	}

	return res, nil
}

// ExtensionCreate creates new Extension record.
func (h *handler) ExtensionCreate(ctx context.Context, b *extension.Extension) error {
	q := `insert into extensions(
		id,
		customer_id,

		name,
		detail,

		endpoint_id,
		aor_id,
		auth_id,

		extension,
		domain_name,

		realm,
		username,
		password,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?, ?,
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

		b.EndpointID,
		b.AORID,
		b.AuthID,

		b.Extension,
		b.DomainName,

		b.Realm,
		b.Username,
		b.Password,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ExtensionCreate. err: %v", err)
	}

	// update the cache
	_ = h.extensionUpdateToCache(ctx, b.ID)

	return nil
}

// extensionGetFromDB returns Extension from the DB.
func (h *handler) extensionGetFromDB(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {

	q := fmt.Sprintf("%s where id = ?", extensionSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.extensionGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionGetFromDB. err: %v", err)
	}

	return res, nil
}

// extensionUpdateToCache gets the extension from the DB and update the cache.
func (h *handler) extensionUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.extensionGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.extensionSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// extensionSetToCache sets the given extension to the cache
func (h *handler) extensionSetToCache(ctx context.Context, e *extension.Extension) error {
	if err := h.cache.ExtensionSet(ctx, e); err != nil {
		return err
	}

	return nil
}

// extensionGetFromCache returns Extension from the cache.
func (h *handler) extensionGetFromCache(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {

	// get from cache
	res, err := h.cache.ExtensionGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// extensionGetByEndpointIDFromCache returns Extension from the cache.
func (h *handler) extensionGetByEndpointIDFromCache(ctx context.Context, endpoint string) (*extension.Extension, error) {

	// get from cache
	res, err := h.cache.ExtensionGetByEndpointID(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// extensionGetByEndpointIDFromCache returns Extension from the cache.
func (h *handler) extensionGetByCustomerIDANDExtensionFromCache(ctx context.Context, customerID uuid.UUID, endpoint string) (*extension.Extension, error) {

	// get from cache
	res, err := h.cache.ExtensionGetByCustomerIDANDExtension(ctx, customerID, endpoint)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ExtensionGet returns extension.
func (h *handler) ExtensionGet(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {

	res, err := h.extensionGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.extensionGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.extensionSetToCache(ctx, res)

	return res, nil
}

// ExtensionGetByEndpointID returns extension of the given extension.
func (h *handler) ExtensionGetByEndpointID(ctx context.Context, endpointID string) (*extension.Extension, error) {

	res, err := h.extensionGetByEndpointIDFromCache(ctx, endpointID)
	if err == nil {
		return res, nil
	}

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			endpoint_id = ?
		order by
			tm_create desc
		limit 1
	`, extensionSelect)

	row, err := h.db.Query(q, endpointID)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionGetByEndpointID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err = h.extensionGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionGetByEndpointID. err: %v", err)
	}

	// set to the cache
	_ = h.extensionSetToCache(ctx, res)

	return res, nil
}

// ExtensionGetByExtension returns extension of the given extension.
func (h *handler) ExtensionGetByExtension(ctx context.Context, customerID uuid.UUID, ext string) (*extension.Extension, error) {

	res, err := h.extensionGetByCustomerIDANDExtensionFromCache(ctx, customerID, ext)
	if err == nil {
		return res, nil
	}

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			customer_id = ?
			and extension = ?
		order by
			tm_create desc
		limit 1
	`, extensionSelect)

	row, err := h.db.Query(q, customerID.Bytes(), ext)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionGetByExtension. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err = h.extensionGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionGetByExtension. err: %v", err)
	}

	// set to the cache
	_ = h.extensionSetToCache(ctx, res)

	return res, nil
}

// ExtensionDelete deletes given extension
func (h *handler) ExtensionDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update extensions set
		tm_delete = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ExtensionDelete. err: %v", err)
	}

	// update the cache
	_ = h.extensionUpdateToCache(ctx, id)

	return nil
}

// ExtensionUpdate updates extension record.
func (h *handler) ExtensionUpdate(ctx context.Context, id uuid.UUID, name string, detail string, password string) error {
	q := `
	update extensions set
		name = ?,
		detail = ?,
		password = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		name,
		detail,
		password,
		h.utilHandler.TimeGetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. ExtensionUpdate. err: %v", err)
	}

	// update the cache
	_ = h.extensionUpdateToCache(ctx, id)

	return nil
}

// ExtensionGets returns list extensions.
func (h *handler) ExtensionGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*extension.Extension, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, extensionSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id":
			q = fmt.Sprintf("%s and customer_id = ?", q)
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
		return nil, fmt.Errorf("could not query. ExtensionGets. err: %v", err)
	}
	defer rows.Close()

	var res []*extension.Extension
	for rows.Next() {
		u, err := h.extensionGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. ExtensionGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}
