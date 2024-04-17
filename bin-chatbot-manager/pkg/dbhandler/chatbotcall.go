package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
)

const (
	chatbotcallSelect = `
	select
		id,
		customer_id,
		chatbot_id,
		chatbot_engine_type,

		activeflow_id,
		reference_type,
		reference_id,

		confbridge_id,
		transcribe_id,

		status,

		gender,
		language,

		messages,

		tm_end,
		tm_create,
		tm_update,
		tm_delete

	from
		chatbotcalls
	`
)

// chatbotcallGetFromRow gets the chatbotcall from the row.
func (h *handler) chatbotcallGetFromRow(row *sql.Rows) (*chatbotcall.Chatbotcall, error) {
	var messages sql.NullString

	res := &chatbotcall.Chatbotcall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.ChatbotID,
		&res.ChatbotEngineType,

		&res.ActiveflowID,
		&res.ReferenceType,
		&res.ReferenceID,

		&res.ConfbridgeID,
		&res.TranscribeID,

		&res.Status,

		&res.Gender,
		&res.Language,

		&messages,

		&res.TMEnd,
		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. chatbotcallGetFromRow. err: %v", err)
	}

	// messages
	if messages.Valid {
		if err := json.Unmarshal([]byte(messages.String), &res.Messages); err != nil {
			return nil, fmt.Errorf("could not unmarshal the chained_call_ids. callGetFromRow. err: %v", err)
		}
	}
	if res.Messages == nil {
		res.Messages = []chatbotcall.Message{}
	}

	return res, nil
}

// ChatbotcallCreate creates a new chatbotcall record.
func (h *handler) ChatbotcallCreate(ctx context.Context, cb *chatbotcall.Chatbotcall) error {
	q := `insert into chatbotcalls(
		id,
		customer_id,
		chatbot_id,
		chatbot_engine_type,

		activeflow_id,
		reference_type,
		reference_id,

		confbridge_id,
		transcribe_id,

		status,

		gender,
		language,

		messages,

		tm_end,
		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?,
		?, ?, ?,
		?, ?,
		?,
		?, ?,
		?,
		?, ?, ?, ?
		)
	`

	if cb.Messages == nil {
		cb.Messages = []chatbotcall.Message{}
	}
	tmpMessages, err := json.Marshal(cb.Messages)
	if err != nil {
		return fmt.Errorf("could not marshal calls. ChatbotcallCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		cb.ID.Bytes(),
		cb.CustomerID.Bytes(),
		cb.ChatbotID.Bytes(),
		cb.ChatbotEngineType,

		cb.ActiveflowID.Bytes(),
		cb.ReferenceType,
		cb.ReferenceID.Bytes(),

		cb.ConfbridgeID.Bytes(),
		cb.TranscribeID.Bytes(),

		cb.Status,

		cb.Gender,
		cb.Language,

		tmpMessages,

		DefaultTimeStamp,
		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ChatbotcallCreate. err: %v", err)
	}

	// update the cache
	_ = h.chatbotcallUpdateToCache(ctx, cb.ID)

	return nil
}

// chatbotallGetFromCache returns chatbotcall from the cache if possible.
func (h *handler) chatbotcallGetFromCache(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error) {

	// get from cache
	res, err := h.cache.ChatbotcallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// chatbotcallGetFromDB gets chatbotcall from the database.
func (h *handler) chatbotcallGetFromDB(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", chatbotcallSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. chatbotcallGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.chatbotcallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. chatbotcallGetFromDB, err: %v", err)
	}

	return res, nil
}

// chatbotcallUpdateToCache gets the chatbotcall from the DB and update the cache.
func (h *handler) chatbotcallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.chatbotcallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.chatbotcallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// chatbotcallSetToCache sets the given chatbotcall to the cache
func (h *handler) chatbotcallSetToCache(ctx context.Context, data *chatbotcall.Chatbotcall) error {
	if err := h.cache.ChatbotcallSet(ctx, data); err != nil {
		return err
	}

	return nil
}

// ChatbotcallGet gets chatbotcall.
func (h *handler) ChatbotcallGet(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error) {

	res, err := h.chatbotcallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.chatbotcallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.chatbotcallSetToCache(ctx, res)

	return res, nil
}

// ChatbotcallGetByReferenceID gets chatbotcall of the given reference_id.
func (h *handler) ChatbotcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*chatbotcall.Chatbotcall, error) {

	tmp, err := h.cache.ChatbotcallGetByReferenceID(ctx, referenceID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf("%s where reference_id = ? order by tm_create desc", chatbotcallSelect)

	row, err := h.db.Query(q, referenceID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatbotcallGetByReferenceID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.chatbotcallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ChatbotcallGetByReferenceID, err: %v", err)
	}

	_ = h.chatbotcallSetToCache(ctx, res)

	return res, nil
}

// ChatbotcallGetByTranscribeID gets chatbotcall of the given transcribe_id.
func (h *handler) ChatbotcallGetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*chatbotcall.Chatbotcall, error) {

	tmp, err := h.cache.ChatbotcallGetByTranscribeID(ctx, transcribeID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf("%s where transcribe_id = ? order by tm_create desc", chatbotcallSelect)

	row, err := h.db.Query(q, transcribeID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatbotcallGetByTranscribeID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.chatbotcallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ChatbotcallGetByTranscribeID, err: %v", err)
	}

	_ = h.chatbotcallSetToCache(ctx, res)

	return res, nil
}

// ChatbotcallUpdateStatusProgressing updates the chatbotcall's status to progressing
func (h *handler) ChatbotcallUpdateStatusProgressing(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) error {
	//prepare
	q := `
	update chatbotcalls set
		status = ?,
		transcribe_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, chatbotcall.StatusProgressing, transcribeID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatbotcallUpdateStatusProgressing. err: %v", err)
	}

	// update the cache
	_ = h.chatbotcallUpdateToCache(ctx, id)

	return nil
}

// ChatbotcallUpdateStatusEnd updates the chatbotcall's status to end
func (h *handler) ChatbotcallUpdateStatusEnd(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update chatbotcalls set
		status = ?,
		transcribe_id = ?,
		tm_end = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, chatbotcall.StatusEnd, uuid.Nil, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatbotcallUpdateStatusEnd. err: %v", err)
	}

	// update the cache
	_ = h.chatbotcallUpdateToCache(ctx, id)

	return nil
}

// ChatbotcallDelete deletes the chatbotcall
func (h *handler) ChatbotcallDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update chatbotcalls set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatbotcallDelete. err: %v", err)
	}

	// update the cache
	_ = h.chatbotcallUpdateToCache(ctx, id)

	return nil
}

// ChatbotcallGets returns a list of chatbotcalls.
func (h *handler) ChatbotcallGets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*chatbotcall.Chatbotcall, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		customer_id = ?
		and tm_create < ?
	`, chatbotcallSelect)

	values := []interface{}{
		customerID.Bytes(),
		token,
	}

	for k, v := range filters {
		switch k {
		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatbotcallGets. err: %v", err)
	}
	defer rows.Close()

	res := []*chatbotcall.Chatbotcall{}
	for rows.Next() {
		u, err := h.chatbotcallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ChatbotcallGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ChatbotcallSetMessages sets the chatbotcall's messages
func (h *handler) ChatbotcallSetMessages(ctx context.Context, id uuid.UUID, messages []chatbotcall.Message) error {

	//prepare
	q := `
	update chatbotcalls set
		messages = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpMessages, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("could not marshal calls. ChatbotcallSetMessages. err: %v", err)
	}

	ts := h.utilHandler.TimeGetCurTime()
	_, err = h.db.Exec(q, tmpMessages, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatbotcallSetMessages. err: %v", err)
	}

	// update the cache
	_ = h.chatbotcallUpdateToCache(ctx, id)

	return nil
}
