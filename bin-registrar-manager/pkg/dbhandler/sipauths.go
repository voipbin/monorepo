package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
)

const (
	sipauthSelect = `
	select
		id,
		reference_type,

		auth_types,
		realm,

		username,
		password,

		allowed_ips,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update
	from
		registrar_sip_auths
	`
)

// sipauthGetFromRow gets the sipauth from the row
func (h *handler) sipauthGetFromRow(row *sql.Rows) (*sipauth.SIPAuth, error) {
	var authTypes sql.NullString
	var allowedIPs sql.NullString

	res := &sipauth.SIPAuth{}
	if err := row.Scan(
		&res.ID,
		&res.ReferenceType,

		&authTypes,
		&res.Realm,

		&res.Username,
		&res.Password,

		&allowedIPs,

		&res.TMCreate,
		&res.TMUpdate,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. sipauthGetFromRow. err: %v", err)
	}

	// AuthTypes
	if authTypes.Valid && authTypes.String != "" {
		if err := json.Unmarshal([]byte(authTypes.String), &res.AuthTypes); err != nil {
			return nil, fmt.Errorf("could not unmarshal the auth_types. sipauthGetFromRow. err: %v", err)
		}
	}
	if res.AuthTypes == nil {
		res.AuthTypes = []sipauth.AuthType{}
	}

	// allowedIPs
	res.AllowedIPs = []string{}
	if allowedIPs.Valid {
		if err := json.Unmarshal([]byte(allowedIPs.String), &res.AllowedIPs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the allowed_ips. sipauthGetFromRow. err: %v", err)
		}
	}
	if res.AllowedIPs == nil {
		res.AllowedIPs = []string{}
	}

	return res, nil
}

// SIPAuthCreate creates new Trunk record.
func (h *handler) SIPAuthCreate(ctx context.Context, t *sipauth.SIPAuth) error {
	q := `insert into registrar_sip_auths(
		id,
		reference_type,

		auth_types,
		realm,

		username,
		password,

		allowed_ips,

		tm_create,
		tm_update
	) values(
		?, ?,
		?, ?,
		?, ?,
		?,
		?, ?
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
		t.ReferenceType,

		authTypes,
		t.Realm,

		t.Username,
		t.Password,

		allowedIps,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. TrunkCreate. err: %v", err)
	}

	return nil
}

// SIPAuthUpdateAll updates all possible sipauth info
func (h *handler) SIPAuthUpdateAll(ctx context.Context, t *sipauth.SIPAuth) error {
	q := `
	update registrar_sip_auths set
		auth_types = ?,
		realm = ?,

		username = ?,
		password = ?,

		allowed_ips = ?,

		tm_update = ?
	where
		id = ?
	`

	tmpAuthTypes, err := json.Marshal(t.AuthTypes)
	if err != nil {
		return fmt.Errorf("could not marshal the authTypes. SIPAuthUpdateAll. err: %v", err)
	}

	tmpAllowedIPs, err := json.Marshal(t.AllowedIPs)
	if err != nil {
		return fmt.Errorf("could not marshal allowedIPs. SIPAuthUpdateAll. err: %v", err)
	}

	_, err = h.db.Exec(q,
		tmpAuthTypes,
		t.Realm,

		t.Username,
		t.Password,

		tmpAllowedIPs,

		h.utilHandler.TimeGetCurTime(),

		t.ID.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. SIPAuthUpdateAll. err: %v", err)
	}

	return nil
}

// SIPAuthDelete deletes the sip auth
func (h *handler) SIPAuthDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	delete from registrar_sip_auths
	where
		id = ?
	`
	_, err := h.db.Exec(q,
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. SIPAuthDelete. err: %v", err)
	}

	return nil
}

// SIPAuthGet returns SIPAuthGet.
func (h *handler) SIPAuthGet(ctx context.Context, id uuid.UUID) (*sipauth.SIPAuth, error) {

	q := fmt.Sprintf("%s where id = ?", sipauthSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. SIPAuthGet. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.sipauthGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. SIPAuthGet. err: %v", err)
	}

	return res, nil
}
