package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/models/aicall"
)

const (
	aicallSelect = `
	select
		id,
		customer_id,

		ai_id,
		ai_engine_type,
		ai_engine_model,
		ai_engine_data,

		activeflow_id,
		reference_type,
		reference_id,

		confbridge_id,
		transcribe_id,

		status,

		gender,
		language,

		tts_streaming_id,
		tts_streaming_pod_id,

		tm_end,
		tm_create,
		tm_update,
		tm_delete

	from
		ai_aicalls
	`
)

// aicallGetFromRow gets the aicall from the row.
func (h *handler) aicallGetFromRow(row *sql.Rows) (*aicall.AIcall, error) {
	var tmpAIEngineData sql.NullString

	res := &aicall.AIcall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.AIID,
		&res.AIEngineType,
		&res.AIEngineModel,
		&tmpAIEngineData,

		&res.ActiveflowID,
		&res.ReferenceType,
		&res.ReferenceID,

		&res.ConfbridgeID,
		&res.TranscribeID,

		&res.Status,

		&res.Gender,
		&res.Language,

		&res.TTSStreamingID,
		&res.TTSStreamingPodID,

		&res.TMEnd,
		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, errors.Wrapf(err, "could not scan the row. aicallGetFromRow")
	}

	if tmpAIEngineData.Valid {
		if err := json.Unmarshal([]byte(tmpAIEngineData.String), &res.AIEngineData); err != nil {
			return nil, fmt.Errorf("could not unmarshal the data. aicallGetFromRow. err: %v", err)
		}
	}
	if res.AIEngineData == nil {
		res.AIEngineData = map[string]any{}
	}

	return res, nil
}

// AIcallCreate creates a new aicall record.
func (h *handler) AIcallCreate(ctx context.Context, cb *aicall.AIcall) error {
	q := `insert into ai_aicalls(
		id,
		customer_id,

		ai_id,
		ai_engine_type,
		ai_engine_model,
		ai_engine_data,

		activeflow_id,
		reference_type,
		reference_id,

		confbridge_id,
		transcribe_id,

		status,

		gender,
		language,
 
		tts_streaming_id,
		tts_streaming_pod_id,

		tm_end,
		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, 
		?, ?, ?, ?,
		?, ?, ?,
		?, ?,
		?,
		?, ?,
		?, ?,
 		?, ?, ?, ?
		)
	`

	tmpAIEngineData, err := json.Marshal(cb.AIEngineData)
	if err != nil {
		return fmt.Errorf("AICreate: Could not marshal the data. err: %v", err)
	}

	_, err = h.db.Exec(q,
		cb.ID.Bytes(),
		cb.CustomerID.Bytes(),

		cb.AIID.Bytes(),
		cb.AIEngineType,
		cb.AIEngineModel,
		tmpAIEngineData,

		cb.ActiveflowID.Bytes(),
		cb.ReferenceType,
		cb.ReferenceID.Bytes(),

		cb.ConfbridgeID.Bytes(),
		cb.TranscribeID.Bytes(),

		cb.Status,

		cb.Gender,
		cb.Language,

		cb.TTSStreamingID.Bytes(),
		cb.TTSStreamingPodID,

		DefaultTimeStamp,
		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. AIcallCreate. err: %v", err)
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, cb.ID)

	return nil
}

// aicallGetFromCache returns aicall from the cache if possible.
func (h *handler) aicallGetFromCache(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {

	// get from cache
	res, err := h.cache.AIcallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// aicallGetFromDB gets aicall from the database.
func (h *handler) aicallGetFromDB(id uuid.UUID) (*aicall.AIcall, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", aicallSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. aicallGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.aicallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. aicallGetFromDB, err: %v", err)
	}

	return res, nil
}

// aicallUpdateToCache gets the aicall from the DB and update the cache.
func (h *handler) aicallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.aicallGetFromDB(id)
	if err != nil {
		return err
	}

	if err := h.aicallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// aicallSetToCache sets the given aicall to the cache
func (h *handler) aicallSetToCache(ctx context.Context, data *aicall.AIcall) error {
	if err := h.cache.AIcallSet(ctx, data); err != nil {
		return err
	}

	return nil
}

// AIcallGet gets aicall.
func (h *handler) AIcallGet(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {

	res, err := h.aicallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.aicallGetFromDB(id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.aicallSetToCache(ctx, res)

	return res, nil
}

// AIcallGetByReferenceID gets aicall of the given reference_id.
func (h *handler) AIcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error) {

	tmp, err := h.cache.AIcallGetByReferenceID(ctx, referenceID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf("%s where reference_id = ? order by tm_create desc", aicallSelect)

	row, err := h.db.Query(q, referenceID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. AIcallGetByReferenceID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.aicallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. AIcallGetByReferenceID, err: %v", err)
	}

	_ = h.aicallSetToCache(ctx, res)

	return res, nil
}

// AIcallGetByStreamingID gets aicall of the given streaming_id.
func (h *handler) AIcallGetByStreamingID(ctx context.Context, streamingID uuid.UUID) (*aicall.AIcall, error) {

	tmp, err := h.cache.AIcallGetByStreamingID(ctx, streamingID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf("%s where tts_streaming_id = ? order by tm_create desc", aicallSelect)

	row, err := h.db.Query(q, streamingID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. AIcallGetByStreamingID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.aicallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. AIcallGetByStreamingID, err: %v", err)
	}

	_ = h.aicallSetToCache(ctx, res)

	return res, nil
}

// AIcallGetByTranscribeID gets aicall of the given transcribe_id.
func (h *handler) AIcallGetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*aicall.AIcall, error) {

	tmp, err := h.cache.AIcallGetByTranscribeID(ctx, transcribeID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf("%s where transcribe_id = ? order by tm_create desc", aicallSelect)

	row, err := h.db.Query(q, transcribeID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. AIcallGetByTranscribeID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.aicallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. AIcallGetByTranscribeID, err: %v", err)
	}

	_ = h.aicallSetToCache(ctx, res)

	return res, nil
}

func (h *handler) aicallUpdateStatus(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID, status aicall.Status) error {
	//prepare
	q := `
		update ai_aicalls set
			status = ?,
			transcribe_id = ?,
			tm_update = ?
		where
			id = ?
		`

	_, err := h.db.Exec(q, status, transcribeID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return errors.Wrapf(err, "could not execute. aicallUpdateStatus")
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, id)

	return nil
}

// AIcallUpdateStatusProgressing updates the aicall's status to progressing
func (h *handler) AIcallUpdateStatusProgressing(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) error {

	return h.aicallUpdateStatus(ctx, id, transcribeID, aicall.StatusProgressing)
}

// AIcallUpdateStatusPausing updates the aicall's status to pausing
func (h *handler) AIcallUpdateStatusPausing(ctx context.Context, id uuid.UUID) error {
	return h.aicallUpdateStatus(ctx, id, uuid.Nil, aicall.StatusPausing)
}

// AIcallUpdateStatusResuming updates the aicall's status to resuming
func (h *handler) AIcallUpdateStatusResuming(ctx context.Context, id uuid.UUID, confbridgeID uuid.UUID) error {
	//prepare
	q := `
	update ai_aicalls set
		status = ?,
		confbridge_id = ?,
 		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, aicall.StatusResuming, confbridgeID.Bytes(), ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AIcallUpdateStatusResuming. err: %v", err)
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, id)

	return nil
}

// AIcallUpdateStatusFinishing updates the aicall's status to finishing
func (h *handler) AIcallUpdateStatusFinishing(ctx context.Context, id uuid.UUID) error {
	return h.aicallUpdateStatus(ctx, id, uuid.Nil, aicall.StatusFinishing)
}

// AIcallUpdateStatusFinished updates the aicall's status to end
func (h *handler) AIcallUpdateStatusFinished(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update ai_aicalls set
		status = ?,
		transcribe_id = ?,
		tm_end = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, aicall.StatusFinished, uuid.Nil.Bytes(), ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AIcallUpdateStatusFinished. err: %v", err)
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, id)

	return nil
}

// AIcallDelete deletes the aicall
func (h *handler) AIcallDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update ai_aicalls set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AIcallDelete. err: %v", err)
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, id)

	return nil
}

// AIcallGets returns a list of aicalls.
func (h *handler) AIcallGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*aicall.AIcall, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, aicallSelect)

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		case "customer_id", "ai_id", "activeflow_id", "reference_id", "confbridge_id", "transcribe_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, uuid.FromStringOrNil(v).Bytes())

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AIcallGets. err: %v", err)
	}
	defer rows.Close()

	res := []*aicall.AIcall{}
	for rows.Next() {
		u, err := h.aicallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. AIcallGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}
