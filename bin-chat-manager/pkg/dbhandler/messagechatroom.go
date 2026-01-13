package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-chat-manager/models/messagechatroom"
)

const (
	messagechatroomTable = "chat_messagechatrooms"
)

// messagechatroomGetFromRow gets the messagechatroom from the row.
func (h *handler) messagechatroomGetFromRow(row *sql.Rows) (*messagechatroom.Messagechatroom, error) {
	res := &messagechatroom.Messagechatroom{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. messagechatroomGetFromRow. err: %v", err)
	}

	return res, nil
}

// MessagechatroomCreate creates a new messagechatroom record
func (h *handler) MessagechatroomCreate(ctx context.Context, m *messagechatroom.Messagechatroom) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	m.TMCreate = now
	m.TMUpdate = commondatabasehandler.DefaultTimeStamp
	m.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(m)
	if err != nil {
		return fmt.Errorf("could not prepare fields. MessagechatroomCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(messagechatroomTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. MessagechatroomCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. MessagechatroomCreate. err: %v", err)
	}

	_ = h.messagechatroomUpdateToCache(ctx, m.ID)

	return nil
}

// messagechatroomUpdateToCache gets the messagechatroom from the DB and update the cache.
func (h *handler) messagechatroomUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.messagechatroomGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.messagechatroomSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// messagechatroomSetToCache sets the given messagechatroom to the cache
func (h *handler) messagechatroomSetToCache(ctx context.Context, m *messagechatroom.Messagechatroom) error {
	if err := h.cache.MessagechatroomSet(ctx, m); err != nil {
		return err
	}

	return nil
}

// messagechatroomGetFromCache returns messagechatroom from the cache if possible.
func (h *handler) messagechatroomGetFromCache(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {
	// get from cache
	res, err := h.cache.MessagechatroomGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// messagechatroomGetFromDB gets the messagechatroom info from the db.
func (h *handler) messagechatroomGetFromDB(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {
	fields := commondatabasehandler.GetDBFields(&messagechatroom.Messagechatroom{})
	query, args, err := squirrel.
		Select(fields...).
		From(messagechatroomTable).
		Where(squirrel.Eq{string(messagechatroom.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. messagechatroomGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. messagechatroomGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. messagechatroomGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.messagechatroomGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MessagechatroomGet returns messagechatroom.
func (h *handler) MessagechatroomGet(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {
	res, err := h.messagechatroomGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.messagechatroomGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.messagechatroomSetToCache(ctx, res)

	return res, nil
}

// MessagechatroomGets returns list of messagechatrooms.
func (h *handler) MessagechatroomGets(ctx context.Context, token string, size uint64, filters map[messagechatroom.Field]any) ([]*messagechatroom.Messagechatroom, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&messagechatroom.Messagechatroom{})
	sb := squirrel.
		Select(fields...).
		From(messagechatroomTable).
		Where(squirrel.Lt{string(messagechatroom.FieldTMCreate): token}).
		OrderBy(string(messagechatroom.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. MessagechatroomGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. MessagechatroomGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessagechatroomGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*messagechatroom.Messagechatroom{}
	for rows.Next() {
		u, err := h.messagechatroomGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. MessagechatroomGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. MessagechatroomGets. err: %v", err)
	}

	return res, nil
}

// MessagechatroomUpdate updates the messagechatroom with the given fields.
func (h *handler) MessagechatroomUpdate(ctx context.Context, id uuid.UUID, fields map[messagechatroom.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[messagechatroom.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("MessagechatroomUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(messagechatroomTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(messagechatroom.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("MessagechatroomUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("MessagechatroomUpdate: exec failed: %w", err)
	}

	_ = h.messagechatroomUpdateToCache(ctx, id)
	return nil
}

// MessagechatroomDelete deletes the given messagechatroom
func (h *handler) MessagechatroomDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[messagechatroom.Field]any{
		messagechatroom.FieldTMUpdate: ts,
		messagechatroom.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("MessagechatroomDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(messagechatroomTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(messagechatroom.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("MessagechatroomDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("MessagechatroomDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.messagechatroomUpdateToCache(ctx, id)

	return nil
}
