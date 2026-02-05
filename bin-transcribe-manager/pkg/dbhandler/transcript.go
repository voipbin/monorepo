package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-transcribe-manager/models/transcript"
)

const (
	transcriptsTable = "transcribe_transcripts"
)

// transcriptGetFromRow gets the transcript from the row.
func (h *handler) transcriptGetFromRow(row *sql.Rows) (*transcript.Transcript, error) {
	res := &transcript.Transcript{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. transcriptGetFromRow. err: %v", err)
	}

	return res, nil
}

// TranscriptCreate creates a new transcript
func (h *handler) TranscriptCreate(ctx context.Context, t *transcript.Transcript) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	t.TMCreate = now
	t.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("could not prepare fields. TranscriptCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(transcriptsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. TranscriptCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. TranscriptCreate. err: %v", err)
	}

	// update the cache
	_ = h.transcriptUpdateToCache(ctx, t.ID)

	return nil
}

// transcriptUpdateToCache gets the transcript from the DB and update the cache.
func (h *handler) transcriptUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.transcriptGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.transcriptSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// transcriptSetToCache sets the transcript to the cache.
func (h *handler) transcriptSetToCache(ctx context.Context, t *transcript.Transcript) error {
	if err := h.cache.TranscriptSet(ctx, t); err != nil {
		return err
	}

	return nil
}

// transcriptGetFromCache gets the transcript from the cache.
func (h *handler) transcriptGetFromCache(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	res, err := h.cache.TranscriptGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TranscriptGet returns transcript.
func (h *handler) TranscriptGet(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	res, err := h.transcriptGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.transcriptGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.transcriptSetToCache(ctx, res)

	return res, nil
}

// transcriptGetFromDB returns transcript from the DB.
func (h *handler) transcriptGetFromDB(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	fields := commondatabasehandler.GetDBFields(&transcript.Transcript{})
	query, args, err := squirrel.
		Select(fields...).
		From(transcriptsTable).
		Where(squirrel.Eq{string(transcript.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. transcriptGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. transcriptGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. transcriptGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.transcriptGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get record. transcriptGetFromDB, err: %v", err)
	}

	return res, nil
}

// TranscriptList returns list of transcripts.
func (h *handler) TranscriptList(ctx context.Context, size uint64, token string, filters map[transcript.Field]any) ([]*transcript.Transcript, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&transcript.Transcript{})
	sb := squirrel.
		Select(fields...).
		From(transcriptsTable).
		Where(squirrel.Lt{string(transcript.FieldTMCreate): token}).
		OrderBy(string(transcript.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. TranscriptGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. TranscriptGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscriptGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*transcript.Transcript{}
	for rows.Next() {
		u, err := h.transcriptGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. TranscriptGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. TranscriptGets. err: %v", err)
	}

	return res, nil
}

// TranscriptDelete deletes the transcript.
func (h *handler) TranscriptDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[transcript.Field]any{
		transcript.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("TranscriptDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(transcriptsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(transcript.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("TranscriptDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("TranscriptDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.transcriptUpdateToCache(ctx, id)

	return nil
}

// TranscriptUpdate updates the transcript with the given fields.
func (h *handler) TranscriptUpdate(ctx context.Context, id uuid.UUID, fields map[transcript.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("TranscriptUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(transcriptsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(transcript.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("TranscriptUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("TranscriptUpdate: exec failed: %w", err)
	}

	_ = h.transcriptUpdateToCache(ctx, id)
	return nil
}
