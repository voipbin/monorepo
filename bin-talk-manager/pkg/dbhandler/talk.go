package dbhandler

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-talk-manager/models/talk"
)

const tableTalks = "talk_chats"

func (h *dbHandler) TalkCreate(ctx context.Context, t *talk.Talk) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	t.TMCreate = now
	t.TMUpdate = now

	fields, err := commondb.PrepareFields(t)
	if err != nil {
		logrus.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	query := sq.Insert(tableTalks).
		SetMap(fields).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		logrus.Errorf("Failed to build query: %v", err)
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		logrus.Errorf("Failed to create talk: %v", err)
		return err
	}

	return nil
}

func (h *dbHandler) TalkGet(ctx context.Context, id uuid.UUID) (*talk.Talk, error) {
	fields := commondb.GetDBFields(&talk.Talk{})

	query := sq.Select(fields...).
		From(tableTalks).
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

	var t talk.Talk
	err = commondb.ScanRow(rows, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (h *dbHandler) TalkList(ctx context.Context, filters map[talk.Field]any, token string, size uint64) ([]*talk.Talk, error) {
	fields := commondb.GetDBFields(&talk.Talk{})

	query := sq.Select(fields...).
		From(tableTalks).
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

	var talks []*talk.Talk
	for rows.Next() {
		var t talk.Talk
		err = commondb.ScanRow(rows, &t)
		if err != nil {
			return nil, err
		}
		talks = append(talks, &t)
	}

	return talks, nil
}

func (h *dbHandler) TalkUpdate(ctx context.Context, id uuid.UUID, fields map[talk.Field]any) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	fields[talk.FieldTMUpdate] = now

	preparedFields, err := commondb.PrepareFields(fields)
	if err != nil {
		logrus.Errorf("Failed to prepare fields: %v", err)
		return err
	}

	query := sq.Update(tableTalks).
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

func (h *dbHandler) TalkDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")

	query := sq.Update(tableTalks).
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
