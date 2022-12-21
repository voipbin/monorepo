package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

const (
	// select query for call get
	transcriptSelect = `
	select
		id,
		customer_id,
		transcribe_id,

		direction,
		message,

		tm_transcript,
		tm_create,
		tm_delete
	from
		transcripts
	`
)

// transcriptGetFromRow gets the transcript from the row.
func (h *handler) transcriptGetFromRow(row *sql.Rows) (*transcript.Transcript, error) {
	res := &transcript.Transcript{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.TranscribeID,

		&res.Direction,
		&res.Message,

		&res.TMTranscript,
		&res.TMCreate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. transcriptGetFromRow. err: %v", err)
	}

	return res, nil
}

// TranscriptCreate creates a new tanscript
func (h *handler) TranscriptCreate(ctx context.Context, t *transcript.Transcript) error {
	q := `insert into transcripts(
		id,
		customer_id,
		transcribe_id,

        direction,
        message,

		tm_transcript,
		tm_create,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?, ?, ?
	)`

	_, err := h.db.Exec(q,
		t.ID.Bytes(),
		t.CustomerID.Bytes(),
		t.TranscribeID.Bytes(),

		t.Direction,
		t.Message,

		t.TMTranscript,
		h.utilHandler.GetCurTime(),
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. TranscriptCreate. err: %v", err)
	}

	// update the cache
	_ = h.transcriptUpdateToCache(ctx, t.ID)

	return nil
}

// transcriptUpdateToCache gets the transcript from the DB and update the cache.
func (h *handler) transcriptUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.TranscriptGetFromDB(ctx, id)
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

	res, err = h.TranscriptGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.transcriptSetToCache(ctx, res)

	return res, nil
}

// TranscriptGetFromDB returns transcript from the DB.
func (h *handler) TranscriptGetFromDB(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", transcriptSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscriptGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.transcriptGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get record. TranscriptGetFromDB, err: %v", err)
	}

	return res, nil
}

// TranscriptGetsByTranscribeID returns a list of transcripts of the given transcribeID.
func (h *handler) TranscriptGetsByTranscribeID(ctx context.Context, transcribeID uuid.UUID) ([]*transcript.Transcript, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			transcribe_id = ?
		order by
			tm_create
		asc
		`, transcriptSelect)

	rows, err := h.db.Query(q, transcribeID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscriptGetsByTranscribeID. err: %v", err)
	}
	defer rows.Close()

	res := []*transcript.Transcript{}
	for rows.Next() {
		u, err := h.transcriptGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. TranscriptGetsByTranscribeID, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}
