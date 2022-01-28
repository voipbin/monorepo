package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

const (
	// select query for call get
	transcribeSelect = `
	select
		id,
		customer_id,
		type,
		reference_id,
		host_id,

		language,
		webhook_uri,
		webhook_method,

		transcripts,

		tm_create,
		tm_update,
		tm_delete

	from
		transcribes
	`
)

// transcribeGetFromRow gets the transcribe from the row.
func (h *handler) transcribeGetFromRow(row *sql.Rows) (*transcribe.Transcribe, error) {
	var transcripts string
	res := &transcribe.Transcribe{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.Type,
		&res.ReferenceID,
		&res.HostID,

		&res.Language,
		&res.WebhookURI,
		&res.WebhookMethod,

		&transcripts,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. transcribeGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(transcripts), &res.Transcripts); err != nil {
		return nil, fmt.Errorf("could not unmarshal the transcripts. transcribeGetFromRow. err: %v", err)
	}
	if res.Transcripts == nil {
		res.Transcripts = []transcribe.Transcript{}
	}

	return res, nil
}

// TranscribeCreate creates a new tanscribe
func (h *handler) TranscribeCreate(ctx context.Context, t *transcribe.Transcribe) error {
	q := `insert into transcribes(
		id,
		customer_id,
		type,
		reference_id,
		host_id,

		language,
		webhook_uri,
		webhook_method,

		transcripts,

		tm_create,
		tm_update,
		tm_delete

	) values(
		?, ?, ?, ?, ?,
		?, ?, ?,
		?,
		?, ?, ?
		)`

	if t.Transcripts == nil {
		t.Transcripts = []transcribe.Transcript{}
	}
	tmpTranscripts, err := json.Marshal(t.Transcripts)
	if err != nil {
		return fmt.Errorf("could not marshal the transcripts. TranscribeCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		t.ID.Bytes(),
		t.CustomerID.Bytes(),
		t.Type,
		t.ReferenceID.Bytes(),
		t.HostID.Bytes(),

		t.Language,
		t.WebhookURI,
		t.WebhookMethod,

		tmpTranscripts,

		t.TMCreate,
		t.TMUpdate,
		t.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute. TranscribeCreate. err: %v", err)
	}

	// update the cache
	h.TranscribeUpdateToCache(ctx, t.ID)

	return nil
}

// TranscribeUpdateToCache gets the transcribe from the DB and update the cache.
func (h *handler) TranscribeUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.TranscribeGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.TranscribeSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// TranscribeSetToCache sets the transcribe to the cache.
func (h *handler) TranscribeSetToCache(ctx context.Context, t *transcribe.Transcribe) error {

	if err := h.cache.TranscribeSet(ctx, t); err != nil {
		return err
	}

	return nil
}

// TranscribeGetFromCache gets the transcribe from the cache.
func (h *handler) TranscribeGetFromCache(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {

	res, err := h.cache.TranscribeGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TranscribeGet returns transcribe.
func (h *handler) TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {

	res, err := h.TranscribeGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.TranscribeGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.TranscribeSetToCache(ctx, res)

	return res, nil
}

// TranscribeGetFromDB returns transcribe from the DB.
func (h *handler) TranscribeGetFromDB(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", transcribeSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. TranscribeGetFromDB. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.transcribeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get transcribe. TranscribeGetFromDB, err: %v", err)
	}

	return res, nil
}

// TranscribeAddTranscript adds the transcript to the transcribe.
func (h *handler) TranscribeAddTranscript(ctx context.Context, id uuid.UUID, t *transcribe.Transcript) error {
	// prepare
	q := `
	update transcribes set
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

	_, err = h.db.Exec(q, m, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. TranscribeAddTranscript. err: %v", err)
	}

	// update the cache
	h.TranscribeUpdateToCache(ctx, id)

	return nil
}
