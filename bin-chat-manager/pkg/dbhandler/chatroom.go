package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-chat-manager/models/chatroom"
)

const (
	chatroomTable = "chat_chatrooms"
)

// chatroomGetFromRow gets the chatroom from the row.
func (h *handler) chatroomGetFromRow(row *sql.Rows) (*chatroom.Chatroom, error) {
	res := &chatroom.Chatroom{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. chatroomGetFromRow. err: %v", err)
	}

	return res, nil
}

// ChatroomCreate creates a new chatroom record
func (h *handler) ChatroomCreate(ctx context.Context, c *chatroom.Chatroom) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = commondatabasehandler.DefaultTimeStamp
	c.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ChatroomCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(chatroomTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ChatroomCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ChatroomCreate. err: %v", err)
	}

	_ = h.chatroomUpdateToCache(ctx, c.ID)

	return nil
}

// chatroomUpdateToCache gets the chatroom from the DB and update the cache.
func (h *handler) chatroomUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.chatroomGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.chatroomSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// chatroomSetToCache sets the given chatroom to the cache
func (h *handler) chatroomSetToCache(ctx context.Context, f *chatroom.Chatroom) error {
	if err := h.cache.ChatroomSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// chatroomGetFromCache returns chatroom from the cache if possible.
func (h *handler) chatroomGetFromCache(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {
	// get from cache
	res, err := h.cache.ChatroomGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// chatroomGetFromDB gets the chatroom info from the db.
func (h *handler) chatroomGetFromDB(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {
	fields := commondatabasehandler.GetDBFields(&chatroom.Chatroom{})
	query, args, err := squirrel.
		Select(fields...).
		From(chatroomTable).
		Where(squirrel.Eq{string(chatroom.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. chatroomGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. chatroomGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. chatroomGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.chatroomGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChatroomGet returns chatroom.
func (h *handler) ChatroomGet(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {
	res, err := h.chatroomGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.chatroomGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.chatroomSetToCache(ctx, res)

	return res, nil
}

// ChatroomGets returns list of chatrooms.
func (h *handler) ChatroomGets(ctx context.Context, token string, size uint64, filters map[chatroom.Field]any) ([]*chatroom.Chatroom, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&chatroom.Chatroom{})
	sb := squirrel.
		Select(fields...).
		From(chatroomTable).
		Where(squirrel.Lt{string(chatroom.FieldTMCreate): token}).
		OrderBy(string(chatroom.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ChatroomGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ChatroomGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatroomGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*chatroom.Chatroom{}
	for rows.Next() {
		u, err := h.chatroomGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ChatroomGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ChatroomGets. err: %v", err)
	}

	return res, nil
}

// ChatroomUpdate updates the chatroom with the given fields.
func (h *handler) ChatroomUpdate(ctx context.Context, id uuid.UUID, fields map[chatroom.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[chatroom.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ChatroomUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(chatroomTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(chatroom.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ChatroomUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ChatroomUpdate: exec failed: %w", err)
	}

	_ = h.chatroomUpdateToCache(ctx, id)
	return nil
}

// ChatroomUpdateBasicInfo updates the basic information.
func (h *handler) ChatroomUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	fields := map[chatroom.Field]any{
		chatroom.FieldName:   name,
		chatroom.FieldDetail: detail,
	}
	return h.ChatroomUpdate(ctx, id, fields)
}

// ChatroomDelete deletes the given chatroom
func (h *handler) ChatroomDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[chatroom.Field]any{
		chatroom.FieldTMUpdate: ts,
		chatroom.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ChatroomDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(chatroomTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(chatroom.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ChatroomDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("ChatroomDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.chatroomUpdateToCache(ctx, id)

	return nil
}

// ChatroomAddParticipantID adds the given participant_id to the participant_ids.
func (h *handler) ChatroomAddParticipantID(ctx context.Context, id, participantID uuid.UUID) error {
	// prepare
	q := `
	update chat_chatrooms set
		participant_ids = json_array_append(
			participant_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, participantID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatroomAddParticipantID. err: %v", err)
	}

	// update the cache
	_ = h.chatroomUpdateToCache(ctx, id)

	return nil
}

// ChatroomRemoveParticipantID removes the given participantID from the participant_ids.
func (h *handler) ChatroomRemoveParticipantID(ctx context.Context, id, participantID uuid.UUID) error {
	// prepare
	q := `
	update chat_chatrooms set
		participant_ids = json_remove(
			participant_ids, replace(
				json_search(
					participant_ids,
					'one',
					?
				),
				'"',
				''
			)
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, participantID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatroomRemoveParticipantID. err: %v", err)
	}

	// update the cache
	_ = h.chatroomUpdateToCache(ctx, id)

	return nil
}
