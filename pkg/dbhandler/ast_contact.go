package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
)

const (
	astContactSelect = `
	select
		id,
		uri,
		expiration_time,
		qualify_frequency,
		outbound_proxy,
		path,
		user_agent,
		qualify_timeout,
		reg_server,
		authenticate_qualify,
		via_addr,
		via_port,
		call_id,
		endpoint,
		prune_on_boot
	from
		ps_contacts
	`
)

func (h *handler) astContactGetFromRow(row *sql.Rows) (*models.AstContact, error) {
	res := &models.AstContact{}
	if err := row.Scan(
		&res.ID,
		&res.URI,
		&res.ExpirationTime,
		&res.QualifyFrequency,
		&res.OutboundProxy,
		&res.Path,
		&res.UserAgent,
		&res.QualifyTimeout,
		&res.RegServer,
		&res.AuthenticateQualify,
		&res.ViaAddr,
		&res.ViaPort,
		&res.CallID,
		&res.Endpoint,
		&res.PruneOnBoot,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. astContactGetFromRow. err: %v", err)
	}

	return res, nil
}

// AstContactsSetToCache sets the given AstContact to the cache
func (h *handler) AstContactsSetToCache(ctx context.Context, ednpoint string, contacts []*models.AstContact) error {
	if err := h.cache.AstContactsSet(ctx, ednpoint, contacts); err != nil {
		return err
	}

	return nil
}

// AstContactsGetFromCache returns AstContact from the cache.
func (h *handler) AstContactsGetFromCache(ctx context.Context, endpoint string) ([]*models.AstContact, error) {

	// get from cache
	res, err := h.cache.AstContactsGet(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AstContactGetsByEndpoint returns AstContact from the DB.
func (h *handler) AstContactGetsByEndpoint(ctx context.Context, endpoint string) ([]*models.AstContact, error) {
	// get from cache
	if res, err := h.AstContactsGetFromCache(ctx, endpoint); err == nil {
		return res, nil
	}

	q := fmt.Sprintf("%s where endpoint = ?", astContactSelect)

	rows, err := h.db.Query(q, endpoint)
	if err != nil {
		return nil, fmt.Errorf("could not query. AstContactGetsByEndpoint. err: %v", err)
	}
	defer rows.Close()

	var res []*models.AstContact
	for rows.Next() {
		u, err := h.astContactGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. AstContactGetsByEndpoint. err: %v", err)
		}
		res = append(res, u)
	}

	// update cache
	h.AstContactsSetToCache(ctx, endpoint, res)

	return res, nil
}
