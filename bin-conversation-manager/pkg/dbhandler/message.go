package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
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

var (
	messagesTable = "conversation_messages"

	messagesFields = []string{ // This now matches the conversation_dbhandler.go style
		string(message.FieldID),
		string(message.FieldCustomerID),
		string(message.FieldConversationID),
		string(message.FieldDirection),
		string(message.FieldStatus),
		string(message.FieldReferenceType),
		string(message.FieldReferenceID),
		string(message.FieldTransactionID),
		string(message.FieldText),
		string(message.FieldMedias),
		string(message.FieldTMCreate),
		string(message.FieldTMUpdate),
		string(message.FieldTMDelete),
	}
)

// messageGetFromRow gets the message from the row.
func (h *handler) messageGetFromRow(row *sql.Rows) (*message.Message, error) {
	var mediasJSON sql.NullString

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
		&mediasJSON,
		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. messageGetFromRow. err: %v", err)
	}

	if !mediasJSON.Valid || mediasJSON.String == "" {
		res.Medias = []media.Media{}
	} else {
		if err := json.Unmarshal([]byte(mediasJSON.String), &res.Medias); err != nil {
			return nil, fmt.Errorf("could not unmarshal Medias. messageGetFromRow. err: %v", err)
		}
	}

	return res, nil
}

// MessageCreate creates a new message record
func (h *handler) MessageCreate(ctx context.Context, msg *message.Message) error {
	now := h.utilHandler.TimeGetCurTime()

	mediasBytes, err := json.Marshal(msg.Medias)
	if err != nil {
		return fmt.Errorf("could not marshal medias. MessageCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(messagesTable).
		Columns(messagesFields...).
		Values(
			msg.ID.Bytes(),
			msg.CustomerID.Bytes(),
			msg.ConversationID.Bytes(),
			msg.Direction,
			msg.Status,
			msg.ReferenceType,
			msg.ReferenceID.Bytes(),
			msg.TransactionID,
			msg.Text,
			mediasBytes,
			now,
			commondatabasehandler.DefaultTimeStamp,
			commondatabasehandler.DefaultTimeStamp,
		).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. MessageCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. MessageCreate. err: %v", err)
	}

	_ = h.messageUpdateToCache(ctx, msg.ID)
	return nil
}

// messageGetFromDB gets the message info from the db.
func (h *handler) messageGetFromDB(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	query, args, err := squirrel.
		Select(messagesFields...).
		From(messagesTable).
		Where(squirrel.Eq{string(message.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. messageGetFromDB. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not query. messageGetFromDB. err: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. messageGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.messageGetFromRow(rows)
	if err != nil {
		// Error wrapping style from conversationGetFromDB
		return nil, errors.Wrapf(err, "could not get message. messageGetFromDB. id: %s. err: %v", id, err)
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

// // MessageGets returns messages.
func (h *handler) MessageGets(ctx context.Context, token string, size uint64, filters map[message.Field]any) ([]*message.Message, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	sb := squirrel.
		Select(messagesFields...).
		From(messagesTable).
		Where(squirrel.Lt{string(message.FieldTMCreate): token}).
		OrderBy(string(message.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	var err error
	sb, err = commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. MessageGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. MessageGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessageGets. err: %v", err)
	}
	defer rows.Close()

	res := []*message.Message{}
	for rows.Next() {
		u, err := h.messageGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. MessageGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. MessageGets. err: %v", err)
	}

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

func (h *handler) MessageUpdate(ctx context.Context, id uuid.UUID, fields map[message.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[message.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	preparedFields := commondatabasehandler.PrepareUpdateFields(fields)
	sb := squirrel.Update(messagesTable).
		SetMap(preparedFields).
		Where(squirrel.Eq{string(message.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("MessageUpdate: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("MessageUpdate: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "MessageUpdate: error fetching RowsAffected")
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.messageUpdateToCache(ctx, id)
	return nil
}

func (h *handler) MessageDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	updateMap := map[string]any{
		string(message.FieldTMUpdate): ts,
		string(message.FieldTMDelete): ts,
	}

	sb := squirrel.Update(messagesTable).
		SetMap(updateMap).
		Where(squirrel.Eq{string(message.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("MessageDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("MessageDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "MessageDelete: error fetching")
	} else if rowsAffected == 0 {
		return ErrNotFound
	}
	_ = h.messageUpdateToCache(ctx, id)
	return nil
}
