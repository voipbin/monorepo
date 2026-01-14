package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
)

const (
	transcribesTable = "transcribe_transcribes"
)

// transcribeGetFromRow gets the transcribe from the row.
func (h *handler) transcribeGetFromRow(row *sql.Rows) (*transcribe.Transcribe, error) {
	res := &transcribe.Transcribe{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. transcribeGetFromRow. err: %v", err)
	}

	// Ensure StreamingIDs is not nil
	if res.StreamingIDs == nil {
		res.StreamingIDs = []uuid.UUID{}
	}

	return res, nil
}

// TranscribeCreate creates a new transcribe
func (h *handler) TranscribeCreate(ctx context.Context, t *transcribe.Transcribe) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	t.TMCreate = now
	t.TMUpdate = commondatabasehandler.DefaultTimeStamp
	t.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Ensure StreamingIDs is not nil
	if t.StreamingIDs == nil {
		t.StreamingIDs = []uuid.UUID{}
	}

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("could not prepare fields. TranscribeCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(transcribesTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. TranscribeCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. TranscribeCreate. err: %v", err)
	}

	// update the cache
	_ = h.transcribeUpdateToCache(ctx, t.ID)

	return nil
}

// transcribeUpdateToCache gets the transcribe from the DB and update the cache.
func (h *handler) transcribeUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.transcribeGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.transcribeSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// transcribeSetToCache sets the transcribe to the cache.
func (h *handler) transcribeSetToCache(ctx context.Context, t *transcribe.Transcribe) error {
	if err := h.cache.TranscribeSet(ctx, t); err != nil {
		return err
	}

	return nil
}

// transcribeGetFromCache gets the transcribe from the cache.
func (h *handler) transcribeGetFromCache(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	res, err := h.cache.TranscribeGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TranscribeGet returns transcribe.
func (h *handler) TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	res, err := h.transcribeGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.transcribeGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.transcribeSetToCache(ctx, res)

	return res, nil
}

// transcribeGetFromDB returns transcribe from the DB.
func (h *handler) transcribeGetFromDB(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	fields := commondatabasehandler.GetDBFields(&transcribe.Transcribe{})
	query, args, err := squirrel.
		Select(fields...).
		From(transcribesTable).
		Where(squirrel.Eq{string(transcribe.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. transcribeGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. transcribeGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. transcribeGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.transcribeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get transcribe. transcribeGetFromDB, err: %v", err)
	}

	return res, nil
}

// TranscribeDelete deletes the transcribe.
func (h *handler) TranscribeDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[transcribe.Field]any{
		transcribe.FieldTMUpdate: ts,
		transcribe.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("TranscribeDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(transcribesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(transcribe.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("TranscribeDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("TranscribeDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.transcribeUpdateToCache(ctx, id)

	return nil
}

// TranscribeGets returns list of transcribes.
func (h *handler) TranscribeGets(ctx context.Context, size uint64, token string, filters map[transcribe.Field]any) ([]*transcribe.Transcribe, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&transcribe.Transcribe{})
	sb := squirrel.
		Select(fields...).
		From(transcribesTable).
		Where(squirrel.Lt{string(transcribe.FieldTMCreate): token}).
		OrderBy(string(transcribe.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. TranscribeGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. TranscribeGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscribeGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*transcribe.Transcribe{}
	for rows.Next() {
		u, err := h.transcribeGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. TranscribeGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. TranscribeGets. err: %v", err)
	}

	return res, nil
}

// TranscribeGetByReferenceIDAndLanguage returns transcribe of the given referenceid and language.
func (h *handler) TranscribeGetByReferenceIDAndLanguage(ctx context.Context, referenceID uuid.UUID, language string) (*transcribe.Transcribe, error) {
	fields := commondatabasehandler.GetDBFields(&transcribe.Transcribe{})

	filters := map[transcribe.Field]any{
		transcribe.FieldReferenceID: referenceID,
		transcribe.FieldLanguage:    language,
		transcribe.FieldDeleted:     false,
	}

	sb := squirrel.
		Select(fields...).
		From(transcribesTable).
		OrderBy(string(transcribe.FieldTMCreate) + " DESC").
		Limit(1).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. TranscribeGetByReferenceIDAndLanguage. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. TranscribeGetByReferenceIDAndLanguage. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscribeGetByReferenceIDAndLanguage. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.transcribeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get transcribe. TranscribeGetByReferenceIDAndLanguage, err: %v", err)
	}

	return res, nil
}

// TranscribeUpdate updates the transcribe with the given fields.
func (h *handler) TranscribeUpdate(ctx context.Context, id uuid.UUID, fields map[transcribe.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[transcribe.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("TranscribeUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(transcribesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(transcribe.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("TranscribeUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("TranscribeUpdate: exec failed: %w", err)
	}

	_ = h.transcribeUpdateToCache(ctx, id)
	return nil
}
