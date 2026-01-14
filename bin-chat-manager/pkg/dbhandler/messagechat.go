package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-chat-manager/models/messagechat"
)

const (
	messagechatTable = "chat_messagechats"
)

// messagechatGetFromRow gets the messagechat from the row.
func (h *handler) messagechatGetFromRow(row *sql.Rows) (*messagechat.Messagechat, error) {
	res := &messagechat.Messagechat{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. messagechatGetFromRow. err: %v", err)
	}

	return res, nil
}

// MessagechatCreate creates a new messagechat record
func (h *handler) MessagechatCreate(ctx context.Context, m *messagechat.Messagechat) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	m.TMCreate = now
	m.TMUpdate = commondatabasehandler.DefaultTimeStamp
	m.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(m)
	if err != nil {
		return fmt.Errorf("could not prepare fields. MessagechatCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(messagechatTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. MessagechatCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. MessagechatCreate. err: %v", err)
	}

	_ = h.messagechatUpdateToCache(ctx, m.ID)

	return nil
}

// messagechatUpdateToCache gets the messagechat from the DB and update the cache.
func (h *handler) messagechatUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.messagechatGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.messagechatSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// messagechatSetToCache sets the given messagechat to the cache
func (h *handler) messagechatSetToCache(ctx context.Context, m *messagechat.Messagechat) error {
	if err := h.cache.MessagechatSet(ctx, m); err != nil {
		return err
	}

	return nil
}

// messagechatGetFromCache returns messagechat from the cache if possible.
func (h *handler) messagechatGetFromCache(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {
	// get from cache
	res, err := h.cache.MessagechatGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// messagechatGetFromDB gets the messagechat info from the db.
func (h *handler) messagechatGetFromDB(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {
	fields := commondatabasehandler.GetDBFields(&messagechat.Messagechat{})
	query, args, err := squirrel.
		Select(fields...).
		From(messagechatTable).
		Where(squirrel.Eq{string(messagechat.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. messagechatGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. messagechatGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. messagechatGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.messagechatGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MessagechatGet returns messagechat.
func (h *handler) MessagechatGet(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {
	res, err := h.messagechatGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.messagechatGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.messagechatSetToCache(ctx, res)

	return res, nil
}

// MessagechatGets returns list of message chat.
func (h *handler) MessagechatGets(ctx context.Context, token string, size uint64, filters map[messagechat.Field]any) ([]*messagechat.Messagechat, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&messagechat.Messagechat{})
	sb := squirrel.
		Select(fields...).
		From(messagechatTable).
		Where(squirrel.Lt{string(messagechat.FieldTMCreate): token}).
		OrderBy(string(messagechat.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. MessagechatGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. MessagechatGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessagechatGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*messagechat.Messagechat{}
	for rows.Next() {
		u, err := h.messagechatGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. MessagechatGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. MessagechatGets. err: %v", err)
	}

	return res, nil
}

// MessagechatUpdate updates the messagechat with the given fields.
func (h *handler) MessagechatUpdate(ctx context.Context, id uuid.UUID, fields map[messagechat.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[messagechat.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("MessagechatUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(messagechatTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(messagechat.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("MessagechatUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("MessagechatUpdate: exec failed: %w", err)
	}

	_ = h.messagechatUpdateToCache(ctx, id)
	return nil
}

// MessagechatDelete deletes the given messagechat
func (h *handler) MessagechatDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[messagechat.Field]any{
		messagechat.FieldTMUpdate: ts,
		messagechat.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("MessagechatDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(messagechatTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(messagechat.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("MessagechatDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("MessagechatDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.messagechatUpdateToCache(ctx, id)

	return nil
}
