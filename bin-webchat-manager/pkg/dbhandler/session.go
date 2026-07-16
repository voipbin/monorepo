package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-webchat-manager/models/session"
)

const (
	webchatSessionsTable = "webchat_sessions"
)

// sessionGetFromRow gets the session from the row.
func (h *handler) sessionGetFromRow(row *sql.Rows) (*session.Session, error) {
	res := &session.Session{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. sessionGetFromRow. err: %v", err)
	}

	return res, nil
}

// SessionCreate creates new session record.
func (h *handler) SessionCreate(ctx context.Context, s *session.Session) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	s.TMLastActivity = now
	s.TMCreate = now
	s.TMUpdate = nil
	s.TMEnd = nil
	s.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(s)
	if err != nil {
		return fmt.Errorf("could not prepare fields. SessionCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(webchatSessionsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. SessionCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. SessionCreate. err: %v", err)
	}

	// update the cache
	_ = h.sessionUpdateToCache(ctx, s.ID)

	return nil
}

// sessionUpdateToCache gets the session from the DB and updates the cache.
func (h *handler) sessionUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.sessionGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.sessionSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// sessionSetToCache sets the given session to the cache
func (h *handler) sessionSetToCache(ctx context.Context, s *session.Session) error {
	if err := h.cache.SessionSet(ctx, s); err != nil {
		return err
	}

	return nil
}

// sessionGetFromCache returns session from the cache.
func (h *handler) sessionGetFromCache(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	res, err := h.cache.SessionGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// sessionGetFromDB returns session from the DB.
func (h *handler) sessionGetFromDB(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	fields := commondatabasehandler.GetDBFields(&session.Session{})
	query, args, err := squirrel.
		Select(fields...).
		From(webchatSessionsTable).
		Where(squirrel.Eq{string(session.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. sessionGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. sessionGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. sessionGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.sessionGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. sessionGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// SessionGet get session from the database.
func (h *handler) SessionGet(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	res, err := h.sessionGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.sessionGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.sessionSetToCache(ctx, res)

	return res, nil
}

// SessionList returns sessions.
func (h *handler) SessionList(ctx context.Context, size uint64, token string, filters map[session.Field]any) ([]*session.Session, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&session.Session{})
	sb := squirrel.
		Select(fields...).
		From(webchatSessionsTable).
		Where(squirrel.Lt{string(session.FieldTMCreate): token}).
		OrderBy(string(session.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. SessionList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. SessionList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. SessionList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*session.Session{}
	for rows.Next() {
		u, err := h.sessionGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. SessionList, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. SessionList. err: %v", err)
	}

	return res, nil
}

// SessionUpdate updates session fields.
func (h *handler) SessionUpdate(ctx context.Context, id uuid.UUID, fields map[session.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[session.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("SessionUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(webchatSessionsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(session.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("SessionUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("SessionUpdate: exec failed: %w", err)
	}

	_ = h.sessionUpdateToCache(ctx, id)
	return nil
}

// SessionDelete soft-deletes the session.
func (h *handler) SessionDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[session.Field]any{
		session.FieldTMUpdate: ts,
		session.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("SessionDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(webchatSessionsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(session.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("SessionDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("SessionDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.sessionUpdateToCache(ctx, id)

	return nil
}
