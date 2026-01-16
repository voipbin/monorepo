package dbhandler

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commondb "monorepo/bin-common-handler/pkg/commondatabasehandler"
	"monorepo/bin-talk-manager/models/talk"
)

const tableTalks = "talk_chats"

func (h *dbHandler) TalkCreate(ctx context.Context, t *talk.Talk) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	t.TMCreate = now
	t.TMUpdate = now

	fields := commondb.PrepareFields(t, []string{"tm_create", "tm_update"})

	query := sq.Insert(tableTalks).
		SetMap(fields).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		log.Errorf("Failed to build query: %v", err)
		return err
	}

	_, err = h.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		log.Errorf("Failed to create talk: %v", err)
		return err
	}

	return nil
}

func (h *dbHandler) TalkGet(ctx context.Context, id uuid.UUID) (*talk.Talk, error) {
	query := sq.Select(talk.GetDBFields()...).
		From(tableTalks).
		Where(sq.Eq{"id": id.Bytes()}).
		PlaceholderFormat(sq.Question)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var t talk.Talk
	row := h.db.QueryRowContext(ctx, sqlQuery, args...)
	err = commondb.ScanRow(row, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (h *dbHandler) TalkList(ctx context.Context, filters map[talk.Field]any, token string, size uint64) ([]*talk.Talk, error) {
	query := sq.Select(talk.GetDBFields()...).
		From(tableTalks).
		OrderBy("tm_create DESC").
		Limit(size).
		PlaceholderFormat(sq.Question)

	// Apply filters
	query = commondb.ApplyFields(query, filters, map[string]bool{
		"deleted": true, // Filter-only field
	})

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
	defer rows.Close()

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

	query := sq.Update(tableTalks).
		SetMap(fields).
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
