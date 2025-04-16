package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
)

const (
	messageSelect = `
	select
		id,
		customer_id,

		conversation_id,
		direction,
		status,

		reference_type,
		reference_id,

		transaction_id,

 		text,
		medias,

		tm_create,
		tm_update,
		tm_delete
	from
		conversation_messages
	`
)

// conversationGetFromRow gets the conversation from the row.
func (h *handler) messageGetFromRow(row *sql.Rows) (*message.Message, error) {
	var medias sql.NullString

	res := &message.Message{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ConversationID,
		&res.Direction,
		&res.Status,

		&res.ReferenceType,
		&res.ReferenceID,

		&res.TransactionID,

		&res.Text,
		&medias,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. messageGetFromRow. err: %v", err)
	}

	if !medias.Valid {
		res.Medias = []media.Media{}
	} else {
		if errMedias := json.Unmarshal([]byte(medias.String), &res.Medias); errMedias != nil {
			return nil, fmt.Errorf("could not unmarshal the Medias. messageGetFromRow. err: %v", errMedias)
		}
	}

	return res, nil
}

// MessageCreate creates a new message record
func (h *handler) MessageCreate(ctx context.Context, m *message.Message) error {

	q := `insert into conversation_messages(
		id,
		customer_id,

		conversation_id,
		direction,
		status,

		reference_type,
		reference_id,

		transaction_id,

 		text,
		medias,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?, ?,
		?, ?,
		?,
 		?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. MessageCreate. err: %v", err)
	}
	defer stmt.Close()

	medias, err := json.Marshal(m.Medias)
	if err != nil {
		return fmt.Errorf("could not marshal the medias. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		m.ID.Bytes(),
		m.CustomerID.Bytes(),

		m.ConversationID.Bytes(),
		m.Direction,
		m.Status,

		m.ReferenceType,
		m.ReferenceID,

		m.TransactionID,

		m.Text,
		medias,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. MessageCreate. err: %v", err)
	}

	_ = h.messageUpdateToCache(ctx, m.ID)

	return nil
}

// messageGetFromDB gets the message info from the db.
func (h *handler) messageGetFromDB(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", messageSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. messageGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. messageGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.messageGetFromRow(row)
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

// messageGetFromCache returns message from the cache.
func (h *handler) messageGetFromCache(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	// get from cache
	res, err := h.cache.MessageGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MessageGet returns message.
func (h *handler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	res, err := h.messageGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.messageGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.messageSetToCache(ctx, res)

	return res, nil
}

// MessageGetsByTransactionID returns message by the transaction_id.
func (h *handler) MessageGetsByTransactionID(ctx context.Context, transactionID string, token string, limit uint64) ([]*message.Message, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and transaction_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, messageSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, transactionID, token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessageGetsByTransactionID. err: %v", err)
	}
	defer rows.Close()

	var res []*message.Message
	for rows.Next() {
		u, err := h.messageGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. MessageGetsByTransactionID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// MessageGetsByConversationID returns list of messages.
func (h *handler) MessageGetsByConversationID(ctx context.Context, conversationID uuid.UUID, token string, limit uint64) ([]*message.Message, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and conversation_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, messageSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, conversationID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessageGetsByConversationID. err: %v", err)
	}
	defer rows.Close()

	var res []*message.Message
	for rows.Next() {
		u, err := h.messageGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. MessageGetsByConversationID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// MessageUpdateStatus updates the message's status.
func (h *handler) MessageUpdateStatus(ctx context.Context, id uuid.UUID, status message.Status) error {

	q := `
	update conversation_messages set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q,
		status,
		ts,
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. MessageUpdateStatus. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, id)

	return nil
}

// MessageDelete deletes the message.
func (h *handler) MessageDelete(ctx context.Context, id uuid.UUID) error {

	q := `
	update conversation_messages set
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
