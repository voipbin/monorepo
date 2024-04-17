package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
)

const (
	astAorSelect = `
	select
		id,

		max_contacts,
		remove_existing,

		default_expiration,
		minimum_expiration,
		maximum_expiration,

		outbound_proxy,
		support_path,

		authenticate_qualify,
		qualify_frequency,
		qualify_timeout,

		contact,
		mailboxes,
		voicemail_extension
	from
		ps_aors
	`
)

// astAORGetFromRow gets the AstAOR from the row
func (h *handler) astAORGetFromRow(row *sql.Rows) (*astaor.AstAOR, error) {
	res := &astaor.AstAOR{}
	if err := row.Scan(
		&res.ID,

		&res.MaxContacts,
		&res.RemoveExisting,

		&res.DefaultExpiration,
		&res.MinimumExpiration,
		&res.MaximumExpiration,

		&res.OutboundProxy,
		&res.SupportPath,

		&res.AuthenticateQualify,
		&res.QualifyFrequency,
		&res.QualifyTimeout,

		&res.Contact,
		&res.Mailboxes,
		&res.VoicemailExtension,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. astAORGetFromRow. err: %v", err)
	}

	return res, nil
}

// AstAORCreate creates new asterisk-aor record.
func (h *handler) AstAORCreate(ctx context.Context, b *astaor.AstAOR) error {
	q := `insert into ps_aors(
		id,

		max_contacts,
		remove_existing,

		default_expiration,
		minimum_expiration,
		maximum_expiration,

		outbound_proxy,
		support_path,

		authenticate_qualify,
		qualify_frequency,
		qualify_timeout,

		contact,
		mailboxes,
		voicemail_extension
	) values(
		?,
		?, ?,
		?, ?, ?,
		?, ?,
		?, ?, ?,
		?, ?, ?
		)
	`

	_, err := h.db.Exec(q,
		b.ID,

		b.MaxContacts,
		b.RemoveExisting,

		b.DefaultExpiration,
		b.MinimumExpiration,
		b.MaximumExpiration,

		b.OutboundProxy,
		b.SupportPath,

		b.AuthenticateQualify,
		b.QualifyFrequency,
		b.QualifyTimeout,

		b.Contact,
		b.Mailboxes,
		b.VoicemailExtension,
	)
	if err != nil {
		return fmt.Errorf("could not execute. AstAORCreate. err: %v", err)
	}

	// update the cache
	_ = h.astAORUpdateToCache(ctx, *b.ID)

	return nil
}

// astAORGetFromDB returns AstAOR from the DB.
func (h *handler) astAORGetFromDB(ctx context.Context, id string) (*astaor.AstAOR, error) {

	q := fmt.Sprintf("%s where id = ?", astAorSelect)

	row, err := h.db.Query(q, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. AstAORGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.astAORGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. AstAORGetFromDB. err: %v", err)
	}

	return res, nil
}

// astAORUpdateToCache gets the AstAOR from the DB and update the cache.
func (h *handler) astAORUpdateToCache(ctx context.Context, id string) error {

	res, err := h.astAORGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.astAORSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// astAORSetToCache sets the given AstAOR to the cache
func (h *handler) astAORSetToCache(ctx context.Context, aor *astaor.AstAOR) error {
	if err := h.cache.AstAORSet(ctx, aor); err != nil {
		return err
	}

	return nil
}

// astAORGetFromCache returns AstAOR from the cache.
func (h *handler) astAORGetFromCache(ctx context.Context, id string) (*astaor.AstAOR, error) {

	// get from cache
	res, err := h.cache.AstAORGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AstAORGet returns AstAOR.
func (h *handler) AstAORGet(ctx context.Context, id string) (*astaor.AstAOR, error) {

	res, err := h.astAORGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.astAORGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.astAORSetToCache(ctx, res)

	return res, nil
}

// AstAORDelete deletes given AstAOR
func (h *handler) AstAORDelete(ctx context.Context, id string) error {

	// delete
	q := `
	delete from ps_aors
	where
		id = ?
	`

	_, err := h.db.Exec(q, id)
	if err != nil {
		return fmt.Errorf("could not execute. AstAORDelete. err: %v", err)
	}

	// delete from the cache
	_ = h.cache.AstAORDel(ctx, id)

	return nil
}
