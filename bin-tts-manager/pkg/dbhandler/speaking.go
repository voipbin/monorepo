package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-tts-manager/models/speaking"
)

const speakingTable = "tts_manager_speaking"

func (h *dbHandler) speakingGetFromRow(row *sql.Rows) (*speaking.Speaking, error) {
	res := &speaking.Speaking{}
	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. speakingGetFromRow. err: %v", err)
	}
	return res, nil
}

func (h *dbHandler) SpeakingCreate(ctx context.Context, s *speaking.Speaking) error {
	s.TMCreate = h.util.TimeNow()

	fields, err := commondatabasehandler.PrepareFields(s)
	if err != nil {
		return fmt.Errorf("could not prepare fields. SpeakingCreate. err: %v", err)
	}

	query, args, err := squirrel.
		Insert(speakingTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. SpeakingCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. SpeakingCreate. err: %v", err)
	}

	return nil
}

func (h *dbHandler) SpeakingGet(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	cols := commondatabasehandler.GetDBFields(&speaking.Speaking{})

	query, args, err := squirrel.
		Select(cols...).
		From(speakingTable).
		Where(squirrel.Eq{string(speaking.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. SpeakingGet. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. SpeakingGet. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("row error. SpeakingGet. err: %v", err)
		}
		return nil, fmt.Errorf("not found")
	}

	return h.speakingGetFromRow(rows)
}

func (h *dbHandler) SpeakingGets(ctx context.Context, token string, size uint64, filters map[speaking.Field]any) ([]*speaking.Speaking, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(&speaking.Speaking{})

	sb := squirrel.
		Select(cols...).
		From(speakingTable).
		Where(squirrel.Lt{string(speaking.FieldTMCreate): token}).
		OrderBy(string(speaking.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. SpeakingGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. SpeakingGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. SpeakingGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var res []*speaking.Speaking
	for rows.Next() {
		s, err := h.speakingGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan row. SpeakingGets. err: %v", err)
		}
		res = append(res, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error. SpeakingGets. err: %v", err)
	}

	return res, nil
}

func (h *dbHandler) SpeakingUpdate(ctx context.Context, id uuid.UUID, fields map[speaking.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[speaking.FieldTMUpdate] = h.util.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. SpeakingUpdate. err: %v", err)
	}

	query, args, err := squirrel.
		Update(speakingTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{"id": id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. SpeakingUpdate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. SpeakingUpdate. err: %v", err)
	}

	return nil
}

func (h *dbHandler) SpeakingDelete(ctx context.Context, id uuid.UUID) error {
	now := h.util.TimeNow()
	fields := map[speaking.Field]any{
		speaking.FieldTMDelete: now,
		speaking.FieldTMUpdate: now,
	}

	return h.SpeakingUpdate(ctx, id, fields)
}
