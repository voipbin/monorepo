package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

const (
	// select query for call get
	transcribeSelect = `
	select
		id,
		customer_id,

		reference_type,
		reference_id,

		status,
		host_id,
		language,
		direction,

		streaming_ids,

		tm_create,
		tm_update,
		tm_delete

	from
		transcribe_transcribes
	`
)

// transcribeGetFromRow gets the transcribe from the row.
func (h *handler) transcribeGetFromRow(row *sql.Rows) (*transcribe.Transcribe, error) {
	var tmpDirection sql.NullString
	var tmpStreamingIDs sql.NullString

	res := &transcribe.Transcribe{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ReferenceType,
		&res.ReferenceID,

		&res.Status,
		&res.HostID,
		&res.Language,
		&tmpDirection,

		&tmpStreamingIDs,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. transcribeGetFromRow. err: %v", err)
	}

	// Direction
	if tmpDirection.Valid {
		res.Direction = transcribe.Direction(tmpDirection.String)
	}

	// StreamingIDs
	if tmpStreamingIDs.Valid {
		if err := json.Unmarshal([]byte(tmpStreamingIDs.String), &res.StreamingIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the recording_ids. callGetFromRow. err: %v", err)
		}
	}
	if res.StreamingIDs == nil {
		res.StreamingIDs = []uuid.UUID{}
	}

	return res, nil
}

// TranscribeCreate creates a new tanscribe
func (h *handler) TranscribeCreate(ctx context.Context, t *transcribe.Transcribe) error {

	if t.StreamingIDs == nil {
		t.StreamingIDs = []uuid.UUID{}
	}
	tmpStreamingIDs, err := json.Marshal(t.StreamingIDs)
	if err != nil {
		return fmt.Errorf("could not marshal the streaming_ids. TranscribeCreate. err: %v", err)
	}

	q := `insert into transcribe_transcribes(
		id,
		customer_id,

		reference_type,
		reference_id,

		status,
		host_id,
		language,
		direction,

		streaming_ids,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?, ?, ?,
		?,
		?, ?, ?
		)`

	_, err = h.db.Exec(q,
		t.ID.Bytes(),
		t.CustomerID.Bytes(),

		t.ReferenceType,
		t.ReferenceID.Bytes(),

		t.Status,
		t.HostID.Bytes(),
		t.Language,
		t.Direction,

		tmpStreamingIDs,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. TranscribeCreate. err: %v", err)
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

	// prepare
	q := fmt.Sprintf("%s where id = ?", transcribeSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscribeGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.transcribeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get transcribe. TranscribeGetFromDB, err: %v", err)
	}

	return res, nil
}

// TranscribeDelete deletes the transcribe.
func (h *handler) TranscribeDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		transcribe_transcribes
	set
		tm_delete = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. TranscribeDelete. err: %v", err)
	}

	// update the cache
	_ = h.transcribeUpdateToCache(ctx, id)

	return nil
}

// TranscribeAddTranscript adds the transcript to the transcribe.
func (h *handler) TranscribeAddTranscript(ctx context.Context, id uuid.UUID, t *transcript.Transcript) error {
	// prepare
	q := `
	update transcribe_transcribes set
		transcripts = json_array_append(
			transcripts,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	m, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("could not marshal the transcripts. TranscribeAddTranscript. err: %v", err)
	}

	_, err = h.db.Exec(q, m, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. TranscribeAddTranscript. err: %v", err)
	}

	// update the cache
	_ = h.transcribeUpdateToCache(ctx, id)

	return nil
}

// TranscribeGets returns list of transcribes.
func (h *handler) TranscribeGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*transcribe.Transcribe, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, transcribeSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "reference_id", "host_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))
	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscribeGets. err: %v", err)
	}
	defer rows.Close()

	res := []*transcribe.Transcribe{}
	for rows.Next() {
		u, err := h.transcribeGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. TranscribeGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// TranscribeGetByReferenceIDAndLanguage returns transcribe of the given referenceid and language.
func (h *handler) TranscribeGetByReferenceIDAndLanguage(ctx context.Context, referenceID uuid.UUID, language string) (*transcribe.Transcribe, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			reference_id = ?
			and language = ?
			and tm_delete >= ?
		order by
			tm_create
		desc limit ?
		`, transcribeSelect)

	row, err := h.db.Query(q, referenceID.Bytes(), language, DefaultTimeStamp)
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscribeGetByReferenceIDAndLanguage. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.transcribeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get transcribe. TranscribeGetByReferenceIDAndLanguage, err: %v", err)
	}

	return res, nil
}

// TranscribeSetStatus sets the transcribe's status
func (h *handler) TranscribeSetStatus(ctx context.Context, id uuid.UUID, status transcribe.Status) error {

	// prepare
	q := `
	update
		transcribe_transcribes
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, status, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not query. TranscribeSetStatus. err: %v", err)
	}

	_ = h.transcribeUpdateToCache(ctx, id)

	return nil
}
