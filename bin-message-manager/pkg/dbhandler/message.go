package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

const (
	messageTable = "message_messages"
)

// messageGetFromRow scans a single row into a Message struct using db tags
func (h *handler) messageGetFromRow(rows *sql.Rows) (*message.Message, error) {
	res := &message.Message{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. messageGetFromRow. err: %v", err)
	}

	// Initialize nil pointers to empty values
	if res.Source == nil {
		res.Source = &commonaddress.Address{}
	}
	if res.Targets == nil {
		res.Targets = []target.Target{}
	}
	if res.Medias == nil {
		res.Medias = []string{}
	}

	return res, nil
}

// MessageCreate creates a new message record.
func (h *handler) MessageCreate(ctx context.Context, m *message.Message) error {
	m.TMCreate = h.utilHandler.TimeGetCurTime()
	m.TMUpdate = DefaultTimeStamp
	m.TMDelete = DefaultTimeStamp

	// prepare fields for insert
	fields, err := commondatabasehandler.PrepareFields(m)
	if err != nil {
		return errors.Wrap(err, "could not prepare fields. MessageCreate")
	}

	query, args, err := sq.Insert(messageTable).SetMap(fields).ToSql()
	if err != nil {
		return errors.Wrap(err, "could not build query. MessageCreate")
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "could not execute. MessageCreate")
	}

	_ = h.messageUpdateToCache(ctx, m.ID)

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
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&message.Message{})

	query, args, err := sq.Select(columns...).
		From(messageTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "could not build query. messageGetFromDB")
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query. messageGetFromDB")
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.messageGetFromRow(rows)
	if err != nil {
		return nil, err
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

	_ = h.messageSetToCache(ctx, res)

	return res, nil
}

// MessageUpdate updates a message with the given fields.
func (h *handler) MessageUpdate(ctx context.Context, id uuid.UUID, fields map[message.Field]any) error {
	// add update timestamp
	fields[message.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return errors.Wrap(err, "could not prepare fields. MessageUpdate")
	}

	query, args, err := sq.Update(messageTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "could not build query. MessageUpdate")
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "could not execute. MessageUpdate")
	}

	_ = h.messageUpdateToCache(ctx, id)

	return nil
}

// MessageUpdateTargets updates the targets and provider name.
func (h *handler) MessageUpdateTargets(ctx context.Context, id uuid.UUID, provider message.ProviderName, targets []target.Target) error {
	fields := map[message.Field]any{
		message.FieldTargets:      targets,
		message.FieldProviderName: provider,
	}

	return h.MessageUpdate(ctx, id, fields)
}

// MessageList returns a list of messages.
func (h *handler) MessageList(ctx context.Context, token string, size uint64, filters map[message.Field]any) ([]*message.Message, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&message.Message{})

	builder := sq.Select(columns...).
		From(messageTable).
		Where("tm_create < ?", token).
		OrderBy("tm_create desc").
		Limit(size)

	// apply filters
	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply filters. MessageList")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "could not build query. MessageList")
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query. MessageList")
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*message.Message{}
	for rows.Next() {
		m, err := h.messageGetFromRow(rows)
		if err != nil {
			return nil, errors.Wrap(err, "could not scan the row. MessageList")
		}

		res = append(res, m)
	}

	return res, nil
}

// MessageDelete deletes the message.
func (h *handler) MessageDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()
	fields := map[message.Field]any{
		message.FieldTMUpdate: ts,
		message.FieldTMDelete: ts,
	}

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return errors.Wrap(err, "could not prepare fields. MessageDelete")
	}

	query, args, err := sq.Update(messageTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "could not build query. MessageDelete")
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "could not execute. MessageDelete")
	}

	_ = h.messageUpdateToCache(ctx, id)

	return nil
}
