package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
)

const (
	trunkSelect = `
	select
		id,
		customer_id,

		name,
		detail,

		domain_name,
		auth_types,

		realm,
		username,
		password,

		allowed_ips,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		registrar_trunks
	`
)

// trunkGetFromRow gets the domain from the row
func (h *handler) trunkGetFromRow(row *sql.Rows) (*trunk.Trunk, error) {
	var authTypes sql.NullString
	var allowedIPs sql.NullString

	res := &trunk.Trunk{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,

		&res.DomainName,
		&authTypes,

		&res.Realm,
		&res.Username,
		&res.Password,

		&allowedIPs,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. trunkGetFromRow. err: %v", err)
	}

	// AuthTypes
	if authTypes.Valid && authTypes.String != "" {
		if err := json.Unmarshal([]byte(authTypes.String), &res.AuthTypes); err != nil {
			return nil, fmt.Errorf("could not unmarshal the auth_types. trunkGetFromRow. err: %v", err)
		}
	}
	if res.AuthTypes == nil {
		res.AuthTypes = []sipauth.AuthType{}
	}

	// allowedIPs
	res.AllowedIPs = []string{}
	if allowedIPs.Valid {
		if err := json.Unmarshal([]byte(allowedIPs.String), &res.AllowedIPs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the allowed_ips. trunkGetFromRow. err: %v", err)
		}
	}
	if res.AllowedIPs == nil {
		res.AllowedIPs = []string{}
	}

	return res, nil
}

// TrunkCreate creates new Trunk record.
func (h *handler) TrunkCreate(ctx context.Context, t *trunk.Trunk) error {
	q := `insert into registrar_trunks(
		id,
		customer_id,

		name,
		detail,

		domain_name,
		auth_types,

		realm,
		username,
		password,

		allowed_ips,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?,
		?,
		?, ?, ?
	)
	`

	authTypes, err := json.Marshal(t.AuthTypes)
	if err != nil {
		return fmt.Errorf("could not marshal auth types. TrunkCreate. err: %v", err)
	}

	allowedIps, err := json.Marshal(t.AllowedIPs)
	if err != nil {
		return fmt.Errorf("could not marshal allowed ips. TrunkCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		t.ID.Bytes(),
		t.CustomerID.Bytes(),

		t.Name,
		t.Detail,

		t.DomainName,
		authTypes,

		t.Realm,
		t.Username,
		t.Password,

		allowedIps,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. TrunkCreate. err: %v", err)
	}

	// update the cache
	_ = h.trunkUpdateToCache(ctx, t.ID)

	return nil
}

// trunkGetFromDB returns Trunk from the DB.
func (h *handler) trunkGetFromDB(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {

	q := fmt.Sprintf("%s where id = ?", trunkSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. trunkGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.trunkGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. trunkGetFromDB. err: %v", err)
	}

	return res, nil
}

// trunkUpdateToCache gets the trunk from the DB and update the cache.
func (h *handler) trunkUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.trunkGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.trunkSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// trunkSetToCache sets the given trunk to the cache
func (h *handler) trunkSetToCache(ctx context.Context, e *trunk.Trunk) error {
	if err := h.cache.TrunkSet(ctx, e); err != nil {
		return err
	}

	return nil
}

// trunkGetFromCache returns trunk from the cache.
func (h *handler) trunkGetFromCache(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {

	// get from cache
	res, err := h.cache.TrunkGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// trunkGetByDomainNameFromCache returns Domain from the cache.
func (h *handler) trunkGetByDomainNameFromCache(ctx context.Context, domainName string) (*trunk.Trunk, error) {

	// get from cache
	res, err := h.cache.TrunkGetByDomainName(ctx, domainName)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// trunkDeleteFromCache deletes Domain from the cache.
//
//nolint:unused // good to have. will use in the future
func (h *handler) trunkDeleteFromCache(ctx context.Context, id uuid.UUID, name string) error {

	// get from cache
	if err := h.cache.TrunkDel(ctx, id, name); err != nil {
		return err
	}

	return nil
}

// TrunkUpdateBasicInfo updates trunk record.
func (h *handler) TrunkUpdateBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	authTypes []sipauth.AuthType,
	username string,
	password string,
	allowedIPs []string,
) error {
	q := `
	update registrar_trunks set
		name = ?,
		detail = ?,
		auth_types = ?,
		username = ?,
		password = ?,
		allowed_ips = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpAuthTypes, err := json.Marshal(authTypes)
	if err != nil {
		return fmt.Errorf("could not marshal the authTypes. TrunkUpdateBasicInfo. err: %v", err)
	}

	tmpAllowedIPs, err := json.Marshal(allowedIPs)
	if err != nil {
		return fmt.Errorf("could not marshal allowedIPs. TrunkUpdateBasicInfo. err: %v", err)
	}

	_, err = h.db.Exec(q,
		name,
		detail,
		tmpAuthTypes,
		username,
		password,
		tmpAllowedIPs,
		h.utilHandler.TimeGetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. TrunkUpdateBasicInfo. err: %v", err)
	}

	// update the cache
	_ = h.trunkUpdateToCache(ctx, id)

	return nil
}

// TrunkGet returns Trunk.
func (h *handler) TrunkGet(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error) {

	res, err := h.trunkGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.trunkGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.trunkSetToCache(ctx, res)

	return res, nil
}

// TrunkGetByDomainName returns Trunk of the given domain name.
func (h *handler) TrunkGetByDomainName(ctx context.Context, domainName string) (*trunk.Trunk, error) {

	res, err := h.trunkGetByDomainNameFromCache(ctx, domainName)
	if err == nil {
		return res, nil
	}

	filters := map[string]string{
		"domain_name": domainName,
		"deleted":     "false",
	}

	tmp, err := h.TrunkGets(ctx, 1, "", filters)
	if err != nil {
		return nil, err
	}

	if len(tmp) == 0 {
		return nil, ErrNotFound
	}

	res = tmp[0]

	// set to the cache
	_ = h.trunkSetToCache(ctx, res)

	return res, nil
}

// TrunkGets returns trunks.
func (h *handler) TrunkGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*trunk.Trunk, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, trunkSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id":
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
		return nil, fmt.Errorf("could not query. TrunkGets. err: %v", err)
	}
	defer rows.Close()

	var res []*trunk.Trunk
	for rows.Next() {
		u, err := h.trunkGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. TrunkGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// TrunkDelete deletes given Trunk
func (h *handler) TrunkDelete(ctx context.Context, id uuid.UUID) error {

	q := `
	update registrar_trunks set
		tm_delete = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. TrunkDelete. err: %v", err)
	}

	_ = h.trunkUpdateToCache(ctx, id)

	return nil
}
