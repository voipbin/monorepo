package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/models/chatbot"
)

const (
	// select query for call get
	chatbotSelect = `
	select
		id,
		customer_id,

		name,
		detail,

		engine_type,
		engine_model,
		engine_data,

		init_prompt,

		tm_create,
		tm_update,
		tm_delete
	from
		chatbot_chatbots
	`
)

// chatbotGetFromRow gets the chatbot from the row.
func (h *handler) chatbotGetFromRow(row *sql.Rows) (*chatbot.Chatbot, error) {
	var tmpEngineData sql.NullString

	res := &chatbot.Chatbot{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,

		&res.EngineType,
		&res.EngineModel,
		&tmpEngineData,

		&res.InitPrompt,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, errors.Wrap(err, "chatbotGetFromRow: Could not scan the row")
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

// ChatbotCreate creates new chatbot record.
func (h *handler) ChatbotCreate(ctx context.Context, c *chatbot.Chatbot) error {
	q := `insert into chatbot_chatbots(
		id,
		customer_id,

		name,
		detail,

		engine_type,
		engine_model,
		engine_data,

		init_prompt,

		tm_create,
		tm_update,
		tm_delete
	) values (
		?, ?,
		?, ?,
		?, ?, ?,
		?,
		?, ?, ?
		)
	`

	tmpEngineData, err := json.Marshal(c.EngineData)
	if err != nil {
		return fmt.Errorf("ChatbotCreate: Could not marshal the data. err: %v", err)
	}
	_, err = h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),

		c.Name,
		c.Detail,

		c.EngineType,
		c.EngineModel,
		tmpEngineData,

		c.InitPrompt,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("ChatbotCreate: Could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.chatbotUpdateToCache(ctx, c.ID)

	return nil
}

// chatbotGetFromCache returns chatbot from the cache.
func (h *handler) chatbotGetFromCache(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error) {

	// get from cache
	res, err := h.cache.ChatbotGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// chatbotGetFromDB returns chatbot from the DB.
func (h *handler) chatbotGetFromDB(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", chatbotSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. chatbotGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.chatbotGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChannelUpdateToCache gets the channel from the DB and update the cache.
func (h *handler) chatbotUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.chatbotGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.chatbotSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// ChannelSetToCache sets the given channel to the cache
func (h *handler) chatbotSetToCache(ctx context.Context, c *chatbot.Chatbot) error {
	if err := h.cache.ChatbotSet(ctx, c); err != nil {
		return err
	}

	return nil
}

// ChatbotGet returns chatbot.
func (h *handler) ChatbotGet(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error) {

	res, err := h.chatbotGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.chatbotGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.chatbotSetToCache(ctx, res)

	return res, nil
}

// ChatbotDelete deletes the chatbot
func (h *handler) ChatbotDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update chatbot_chatbots set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatbotDelete. err: %v", err)
	}

	// update the cache
	_ = h.chatbotUpdateToCache(ctx, id)

	return nil
}

// ChatbotGets returns a list of chatbots.
func (h *handler) ChatbotGets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*chatbot.Chatbot, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		customer_id = ?
		and tm_create < ?
	`, chatbotSelect)

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
		return nil, fmt.Errorf("could not query. ChatbotGets. err: %v", err)
	}
	defer rows.Close()

	res := []*chatbot.Chatbot{}
	for rows.Next() {
		u, err := h.chatbotGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ChatbotGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ChatbotSetInfo sets the chatbot info
func (h *handler) ChatbotSetInfo(ctx context.Context, id uuid.UUID, name string, detail string, engineType chatbot.EngineType, engineModel chatbot.EngineModel, engineData map[string]any, initPrompt string) error {
	q := `
	update chatbot_chatbots set
		name = ?,
		detail = ?,
		engine_type = ?,
		engine_model = ?,
		engine_data = ?,
		init_prompt = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpEngineData, err := json.Marshal(engineData)
	if err != nil {
		return errors.Wrapf(err, "ChatbotSetInfo: Could not marshal the data")
	}

	ts := h.utilHandler.TimeGetCurTime()
	_, err = h.db.Exec(q, name, detail, engineType, engineModel, tmpEngineData, initPrompt, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatbotSetInfo. err: %v", err)
	}

	// update the cache
	_ = h.chatbotUpdateToCache(ctx, id)

	return nil
}
