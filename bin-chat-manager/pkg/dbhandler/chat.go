package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-chat-manager/models/chat"
)

const (
	chatTable = "chat_chats"
)

// chatGetFromRow gets the chat from the row.
func (h *handler) chatGetFromRow(row *sql.Rows) (*chat.Chat, error) {
	res := &chat.Chat{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. chatGetFromRow. err: %v", err)
	}

	return res, nil
}

// ChatCreate creates a new chat record
func (h *handler) ChatCreate(ctx context.Context, c *chat.Chat) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = commondatabasehandler.DefaultTimeStamp
	c.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Sort participant IDs before storing
	c.ParticipantIDs = sortUUIDs(c.ParticipantIDs)

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ChatCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(chatTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ChatCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ChatCreate. err: %v", err)
	}

	_ = h.chatUpdateToCache(ctx, c.ID)

	return nil
}

// chatUpdateToCache gets the chat from the DB and update the cache.
func (h *handler) chatUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.chatGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.chatSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// chatSetToCache sets the given chat to the cache
func (h *handler) chatSetToCache(ctx context.Context, c *chat.Chat) error {
	if err := h.cache.ChatSet(ctx, c); err != nil {
		return err
	}

	return nil
}

// chatGetFromCache returns chat from the cache if possible.
func (h *handler) chatGetFromCache(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	// get from cache
	res, err := h.cache.ChatGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// chatGetFromDB gets the chat info from the db.
func (h *handler) chatGetFromDB(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	fields := commondatabasehandler.GetDBFields(&chat.Chat{})
	query, args, err := squirrel.
		Select(fields...).
		From(chatTable).
		Where(squirrel.Eq{string(chat.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. chatGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. chatGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. chatGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.chatGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChatGet returns chat.
func (h *handler) ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	res, err := h.chatGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.chatGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.chatSetToCache(ctx, res)

	return res, nil
}

// ChatGets returns list of chats.
func (h *handler) ChatGets(ctx context.Context, token string, size uint64, filters map[chat.Field]any) ([]*chat.Chat, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&chat.Chat{})
	sb := squirrel.
		Select(fields...).
		From(chatTable).
		Where(squirrel.Lt{string(chat.FieldTMCreate): token}).
		OrderBy(string(chat.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	// Handle participant_ids filter separately (special case)
	participantIDsFilter := ""
	for k, v := range filters {
		if k == chat.FieldParticipantIDs {
			if strVal, ok := v.(string); ok {
				participantIDsFilter = h.chatFilterParseParticipantIDs(strVal)
				delete(filters, k)
			}
			break
		}
	}

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ChatGets. err: %v", err)
	}

	// Apply participant_ids filter if present
	if participantIDsFilter != "" {
		sb = sb.Where("participant_ids = json_array(?)", participantIDsFilter)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ChatGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*chat.Chat{}
	for rows.Next() {
		u, err := h.chatGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ChatGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ChatGets. err: %v", err)
	}

	return res, nil
}

func (h *handler) chatFilterParseParticipantIDs(participantIDs string) string {
	if participantIDs == "" {
		return ""
	}

	ids := strings.Split(participantIDs, ",")
	sort.Strings(ids)

	res := ""
	for i, id := range ids {
		if i == 0 {
			res = fmt.Sprintf(`"%s"`, id)
		} else {
			res = fmt.Sprintf(`%s,"%s"`, res, id)
		}
	}
	res = fmt.Sprintf(`[%s]`, res)

	return res
}

// ChatUpdate updates the chat with the given fields.
func (h *handler) ChatUpdate(ctx context.Context, id uuid.UUID, fields map[chat.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[chat.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	// Sort participant IDs if they are being updated
	if participantIDs, ok := fields[chat.FieldParticipantIDs]; ok {
		if ids, ok := participantIDs.([]uuid.UUID); ok {
			fields[chat.FieldParticipantIDs] = sortUUIDs(ids)
		}
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ChatUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(chatTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(chat.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ChatUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ChatUpdate: exec failed: %w", err)
	}

	_ = h.chatUpdateToCache(ctx, id)
	return nil
}

// ChatUpdateBasicInfo updates the basic information.
func (h *handler) ChatUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	fields := map[chat.Field]any{
		chat.FieldName:   name,
		chat.FieldDetail: detail,
	}
	return h.ChatUpdate(ctx, id, fields)
}

// ChatDelete deletes the given chat
func (h *handler) ChatDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[chat.Field]any{
		chat.FieldTMUpdate: ts,
		chat.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ChatDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(chatTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(chat.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ChatDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("ChatDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.chatUpdateToCache(ctx, id)

	return nil
}

// ChatUpdateRoomOwnerID updates the chat's owner_id.
func (h *handler) ChatUpdateRoomOwnerID(ctx context.Context, id uuid.UUID, roomOwnerID uuid.UUID) error {
	fields := map[chat.Field]any{
		chat.FieldRoomOwnerID: roomOwnerID,
	}
	return h.ChatUpdate(ctx, id, fields)
}

// ChatUpdateParticipantID updates the given participant_id to the participant_ids.
func (h *handler) ChatUpdateParticipantID(ctx context.Context, id uuid.UUID, participantIDs []uuid.UUID) error {
	fields := map[chat.Field]any{
		chat.FieldParticipantIDs: participantIDs,
	}
	return h.ChatUpdate(ctx, id, fields)
}
