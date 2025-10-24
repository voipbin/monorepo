package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/models/ai"
)

const (
	// select query for call get
	aiSelect = `
	select
		id,
		customer_id,

		name,
		detail,

		engine_type,
		engine_model,
		engine_data,
		engine_key,

		init_prompt,

		tts_type,
		tts_voice_id,

		stt_type,

		tm_create,
		tm_update,
		tm_delete
	from
		ai_ais
	`
)

// aiGetFromRow gets the ai from the row.
func (h *handler) aiGetFromRow(row *sql.Rows) (*ai.AI, error) {
	var tmpEngineData sql.NullString

	res := &ai.AI{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,

		&res.EngineType,
		&res.EngineModel,
		&tmpEngineData,
		&res.EngineKey,

		&res.InitPrompt,

		&res.TTSType,
		&res.TTSVoiceID,

		&res.STTType,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, errors.Wrap(err, "aiGetFromRow: Could not scan the row")
	}

	if tmpEngineData.Valid {
		if err := json.Unmarshal([]byte(tmpEngineData.String), &res.EngineData); err != nil {
			return nil, fmt.Errorf("could not unmarshal the data. callGetFromRow. err: %v", err)
		}
	}
	if res.EngineData == nil {
		res.EngineData = map[string]any{}
	}

	return res, nil
}

// AICreate creates new ai record.
func (h *handler) AICreate(ctx context.Context, c *ai.AI) error {
	q := `insert into ai_ais(
		id,
		customer_id,

		name,
		detail,

		engine_type,
		engine_model,
		engine_data,
		engine_key,

		init_prompt,

		tts_type,
		tts_voice_id,

		stt_type,

		tm_create,
		tm_update,
		tm_delete
	) values (
		?, ?,
		?, ?,
		?, ?, ?, ?,
		?,
		?, ?,
		?,
		?, ?, ?
		)
	`

	tmpEngineData, err := json.Marshal(c.EngineData)
	if err != nil {
		return fmt.Errorf("AICreate: Could not marshal the data. err: %v", err)
	}
	_, err = h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),

		c.Name,
		c.Detail,

		c.EngineType,
		c.EngineModel,
		tmpEngineData,
		c.EngineKey,

		c.InitPrompt,

		c.TTSType,
		c.TTSVoiceID,

		c.STTType,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("AICreate: Could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, c.ID)

	return nil
}

// aiGetFromCache returns ai from the cache.
func (h *handler) aiGetFromCache(ctx context.Context, id uuid.UUID) (*ai.AI, error) {

	// get from cache
	res, err := h.cache.AIGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// aiGetFromDB returns ai from the DB.
func (h *handler) aiGetFromDB(ctx context.Context, id uuid.UUID) (*ai.AI, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", aiSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. aiGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.aiGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChannelUpdateToCache gets the channel from the DB and update the cache.
func (h *handler) aiUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.aiGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.aiSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// ChannelSetToCache sets the given channel to the cache
func (h *handler) aiSetToCache(ctx context.Context, c *ai.AI) error {
	if err := h.cache.AISet(ctx, c); err != nil {
		return err
	}

	return nil
}

// AIGet returns ai.
func (h *handler) AIGet(ctx context.Context, id uuid.UUID) (*ai.AI, error) {

	res, err := h.aiGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.aiGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.aiSetToCache(ctx, res)

	return res, nil
}

// AIDelete deletes the ai
func (h *handler) AIDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update ai_ais set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AIDelete. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, id)

	return nil
}

// AIGets returns a list of ais.
func (h *handler) AIGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*ai.AI, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, aiSelect)

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

		case "customer_id":
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
		return nil, fmt.Errorf("could not query. AIGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*ai.AI{}
	for rows.Next() {
		u, err := h.aiGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. AIGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// AISetInfo sets the ai info
func (h *handler) AISetInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	engineType ai.EngineType,
	engineModel ai.EngineModel,
	engineData map[string]any,
	engineKey string,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
) error {
	q := `
	update ai_ais set
		name = ?,
		detail = ?,
		engine_type = ?,
		engine_model = ?,
		engine_data = ?,
		engine_key = ?,
		init_prompt = ?,
		tts_type = ?,
		tts_voice_id = ?,
		stt_type = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpEngineData, err := json.Marshal(engineData)
	if err != nil {
		return errors.Wrapf(err, "AISetInfo: Could not marshal the data")
	}

	ts := h.utilHandler.TimeGetCurTime()
	_, err = h.db.Exec(q, name, detail, engineType, engineModel, tmpEngineData, engineKey, initPrompt, ttsType, ttsVoiceID, sttType, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AISetInfo. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, id)

	return nil
}
