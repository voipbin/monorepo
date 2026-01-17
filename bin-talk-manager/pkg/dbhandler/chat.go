package dbhandler

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/chat"
)

const tableChats = "chat_chats"

func (h *dbHandler) ChatCreate(ctx context.Context, t *chat.Chat) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	t.TMCreate = now
	t.TMUpdate = now

	fields, err := commondb.PrepareFields(t)
	if err != nil {
		logrus.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	query := sq.Insert(tableChats).
		SetMap(fields).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		logrus.Errorf("Failed to build query: %v", err)
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		logrus.Errorf("Failed to create chat: %v", err)
		return err
	}

	return nil
}

func (h *dbHandler) ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
	fields := commondb.GetDBFields(&chat.Chat{})

	query := sq.Select(fields...).
		From(tableChats).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := h.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			logrus.Errorf("Failed to close rows: %v", closeErr)
		}
	}()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	var t chat.Chat
	err = commondb.ScanRow(rows, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (h *dbHandler) ChatList(ctx context.Context, filters map[chat.Field]any, token string, size uint64) ([]*chat.Chat, error) {
	fields := commondb.GetDBFields(&chat.Chat{})

	query := sq.Select(fields...).
		From(tableChats).
		OrderBy("tm_create DESC").
		Limit(size).
		PlaceholderFormat(sq.Question)

	// Apply filters
	query, err := commondb.ApplyFields(query, filters)
	if err != nil {
		logrus.Errorf("Failed to apply filters: %v", err)
		return nil, err
	}

	// Apply pagination token
	if token != "" {
		query = query.Where(sq.Lt{"tm_create": token})
	}

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := h.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			logrus.Errorf("Failed to close rows: %v", closeErr)
		}
	}()

	var talks []*chat.Chat
	for rows.Next() {
		var t chat.Chat
		err = commondb.ScanRow(rows, &t)
		if err != nil {
			return nil, err
		}
		talks = append(talks, &t)
	}

	return talks, nil
}

func (h *dbHandler) TalkUpdate(ctx context.Context, id uuid.UUID, fields map[chat.Field]any) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	fields[chat.FieldTMUpdate] = now

	preparedFields, err := commondb.PrepareFields(fields)
	if err != nil {
		logrus.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	query := sq.Update(tableChats).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	return err
}

func (h *dbHandler) ChatDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")

	query := sq.Update(tableChats).
		Set("tm_delete", now).
		Set("tm_update", now).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	return err
}
