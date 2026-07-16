package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-webchat-manager/models/message"
)

const (
	webchatMessagesTable = "webchat_messages"
)

// messageGetFromRow gets the message from the row.
func (h *handler) messageGetFromRow(row *sql.Rows) (*message.Message, error) {
	res := &message.Message{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. messageGetFromRow. err: %v", err)
	}

	return res, nil
}

// MessageCreate creates new message record.
func (h *handler) MessageCreate(ctx context.Context, m *message.Message) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	m.TMCreate = now
	m.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(m)
	if err != nil {
		return fmt.Errorf("could not prepare fields. MessageCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(webchatMessagesTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. MessageCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. MessageCreate. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, m.ID)

	return nil
}

// messageUpdateToCache gets the message from the DB and updates the cache.
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

// messageSetToCache sets the given message to the cache
func (h *handler) messageSetToCache(ctx context.Context, m *message.Message) error {
	if err := h.cache.MessageSet(ctx, m); err != nil {
		return err
	}

	return nil
}

// messageGetFromCache returns message from the cache.
func (h *handler) messageGetFromCache(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	res, err := h.cache.MessageGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// messageGetFromDB returns message from the DB.
func (h *handler) messageGetFromDB(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	fields := commondatabasehandler.GetDBFields(&message.Message{})
	query, args, err := squirrel.
		Select(fields...).
		From(webchatMessagesTable).
		Where(squirrel.Eq{string(message.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. messageGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. messageGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. messageGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.messageGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. messageGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// MessageGet get message from the database.
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

// MessageList returns messages.
func (h *handler) MessageList(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&message.Message{})
	sb := squirrel.
		Select(fields...).
		From(webchatMessagesTable).
		Where(squirrel.Lt{string(message.FieldTMCreate): token}).
		OrderBy(string(message.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. MessageList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. MessageList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessageList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*message.Message{}
	for rows.Next() {
		u, err := h.messageGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. MessageList, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. MessageList. err: %v", err)
	}

	return res, nil
}

// MessageDelete soft-deletes the message.
func (h *handler) MessageDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[message.Field]any{
		message.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("MessageDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(webchatMessagesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(message.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("MessageDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("MessageDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, id)

	return nil
}
