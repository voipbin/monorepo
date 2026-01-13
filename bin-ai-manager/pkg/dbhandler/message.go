package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/message"
)

const (
	messageTable = "ai_messages"
)

// MessageCreate creates a new message record.
func (h *handler) MessageCreate(ctx context.Context, c *message.Message) error {
	c.TMCreate = h.utilHandler.TimeGetCurTime()
	c.TMDelete = DefaultTimeStamp

	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("MessageCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(messageTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("MessageCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("MessageCreate: could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, c.ID)

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
	cols := commondatabasehandler.GetDBFields(message.Message{})

	query, args, err := sq.Select(cols...).
		From(messageTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("messageGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("messageGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &message.Message{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("messageGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
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

// messageSetToCache sets the given message to the cache.
func (h *handler) messageSetToCache(ctx context.Context, c *message.Message) error {
	if err := h.cache.MessageSet(ctx, c); err != nil {
		return err
	}

	return nil
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

	// set to the cache
	_ = h.messageSetToCache(ctx, res)

	return res, nil
}

// MessageDelete deletes the message.
func (h *handler) MessageDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	query, args, err := sq.Update(messageTable).
		SetMap(map[string]any{
			"tm_delete": ts,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("MessageDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("MessageDelete: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, id)

	return nil
}

// MessageGets returns a list of messages.
func (h *handler) MessageGets(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(message.Message{})

	builder := sq.Select(cols...).
		From(messageTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("MessageGets: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("MessageGets: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("MessageGets: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*message.Message{}
	for rows.Next() {
		u := &message.Message{}
		if err := commondatabasehandler.ScanRow(rows, u); err != nil {
			return nil, fmt.Errorf("MessageGets: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}
