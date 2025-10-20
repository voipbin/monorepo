package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

const (
	messageSelect = `
	select
		id,
		customer_id,
		type,

		source,
		targets,

		provider_name,
		provider_reference_id,

		text,
		medias,
		direction,

		tm_create,
		tm_update,
		tm_delete

	from
		message_messages
	`
)

// messageGetFromRow gets the message from the row.
func (h *handler) messageGetFromRow(row *sql.Rows) (*message.Message, error) {

	var source string
	var targets string
	var medias string

	res := &message.Message{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.Type,

		&source,
		&targets,

		&res.ProviderName,
		&res.ProviderReferenceID,

		&res.Text,
		&medias,
		&res.Direction,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. messageGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(source), &res.Source); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. messageGetFromRow. err: %v", err)
	}
	if res.Source == nil {
		res.Source = &commonaddress.Address{}
	}

	if err := json.Unmarshal([]byte(targets), &res.Targets); err != nil {
		return nil, fmt.Errorf("could not unmarshal the targets. messageGetFromRow. err: %v", err)
	}
	if res.Targets == nil {
		res.Targets = []target.Target{}
	}

	if err := json.Unmarshal([]byte(medias), &res.Medias); err != nil {
		return nil, fmt.Errorf("could not unmarshal the targets. messageGetFromRow. err: %v", err)
	}
	if res.Medias == nil {
		res.Medias = []string{}
	}

	return res, nil
}

// MessageCreate creates a new message record.
func (h *handler) MessageCreate(ctx context.Context, n *message.Message) error {
	q := `insert into message_messages(
		id,
		customer_id,
		type,

		source,
		targets,

		provider_name,
		provider_reference_id,

		text,
		medias,
		direction,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?, ?,
		?, ?, ?,
		?, ?, ?
		)`

	tmpSource, err := json.Marshal(n.Source)
	if err != nil {
		return fmt.Errorf("could not marshal source. MessageCreate. err: %v", err)
	}

	tmpTargets, err := json.Marshal(n.Targets)
	if err != nil {
		return fmt.Errorf("could not marshal targets. MessageCreate. err: %v", err)
	}

	tmpMedias, err := json.Marshal(n.Medias)
	if err != nil {
		return fmt.Errorf("could not marshal medias. MessageCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		n.ID.Bytes(),
		n.CustomerID.Bytes(),
		n.Type,

		tmpSource,
		tmpTargets,

		n.ProviderName,
		n.ProviderReferenceID,

		n.Text,
		tmpMedias,
		n.Direction,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. MessageCreate. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, n.ID)

	return nil
}

// messageGetFromCache returns message from the cache.
func (h *handler) messageGetFromCache(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	// get from cache
	res, err := h.cache.MessageGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// messageSetToCache sets the given message to the cache
func (h *handler) messageSetToCache(ctx context.Context, m *message.Message) error {
	if err := h.cache.MessageSet(ctx, m); err != nil {
		return err
	}

	return nil
}

// messageUpdateToCache gets the message from the DB and update the cache.
func (h *handler) messageUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.messageGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.messageSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// messageGetFromDB returns Message info from the DB.
func (h *handler) messageGetFromDB(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", messageSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. messageGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.messageGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get message. messageGetFromDB, err: %v", err)
	}

	return res, nil
}

// MessageGet returns Message.
func (h *handler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	res, err := h.messageGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.messageGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.messageSetToCache(ctx, res)

	return res, nil
}

// NumberUpdateBasicInfo updates flow id.
func (h *handler) MessageUpdateTargets(ctx context.Context, id uuid.UUID, provider message.ProviderName, targets []target.Target) error {

	q := `
	update message_messages set
		targets = ?,
		provider_name = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpTargets, err := json.Marshal(targets)
	if err != nil {
		return fmt.Errorf("could not marshal targets. MessageUpdateTargets. err: %v", err)
	}

	_, err = h.db.Exec(q,
		tmpTargets,
		provider,
		h.utilHandler.TimeGetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. MessageUpdateTargets. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, id)

	return nil
}

// MessageGets returns a list of numbers.
func (h *handler) MessageGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*message.Message, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			customer_id = ?
			and tm_create < ?
			and tm_delete >= ?
		order by
			tm_create
		desc limit ?
		`, messageSelect)

	rows, err := h.db.Query(q, customerID.Bytes(), token, DefaultTimeStamp, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessageGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*message.Message{}
	for rows.Next() {
		u, err := h.messageGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. MessageGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// MessageDelete deletes the message.
func (h *handler) MessageDelete(ctx context.Context, id uuid.UUID) error {

	q := `
	update message_messages set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q,
		ts,
		ts,
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. MessageDelete. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, id)

	return nil
}
