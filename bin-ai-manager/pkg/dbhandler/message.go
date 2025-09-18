package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"monorepo/bin-ai-manager/models/message"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

const (
	// select query for message get
	messageSelect = `
	select
		id,
		customer_id,
		aicall_id,
		
		direction,
		role,
		content,

		tool_calls,
		tool_call_id,

		tm_create,
		tm_delete
	from
		ai_messages
	`
)

// messageGetFromRow gets the message from the row.
func (h *handler) messageGetFromRow(row *sql.Rows) (*message.Message, error) {
	var tmpToolCalls sql.NullString

	res := &message.Message{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.AIcallID,

		&res.Direction,
		&res.Role,
		&res.Content,

		&tmpToolCalls,
		&res.ToolCallID,

		&res.TMCreate,
		&res.TMDelete,
	); err != nil {
		return nil, errors.Wrap(err, "messageGetFromRow: Could not scan the row")
	}

	if tmpToolCalls.Valid {
		if err := json.Unmarshal([]byte(tmpToolCalls.String), &res.ToolCalls); err != nil {
			return nil, fmt.Errorf("could not unmarshal the data. messageGetFromRow. err: %v", err)
		}
	}
	if res.ToolCalls == nil {
		res.ToolCalls = []message.ToolCall{}
	}

	return res, nil
}

// MessageCreate creates a new message record.
func (h *handler) MessageCreate(ctx context.Context, c *message.Message) error {
	q := `insert into ai_messages(
		id,
		customer_id,
		aicall_id,

		direction,
		role,
		content,

		tool_calls,
		tool_call_id,

		tm_create,
		tm_delete
	) values (
		?, ?, ?,
		?, ?, ?,
		?, ?,
		?, ?
		)
	`

	tmpToolCalls, err := json.Marshal(c.ToolCalls)
	if err != nil {
		return fmt.Errorf("MessageCreate: Could not marshal the data. err: %v", err)
	}

	_, err = h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),
		c.AIcallID.Bytes(),

		c.Direction,
		c.Role,
		c.Content,

		tmpToolCalls,
		c.ToolCallID,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("MessageCreate: Could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, c.ID)

	return nil
}

// messageGetFromCache returns message from the cache.
func (h *handler) messageGetFromCache(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	// get from cache
	res, err := h.cache.MessageGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// messageGetFromDB returns message from the DB.
func (h *handler) messageGetFromDB(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", messageSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. messageGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.messageGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// messageUpdateToCache gets the message from the DB and updates the cache.
func (h *handler) messageUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.messageGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.messageSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// messageSetToCache sets the given message to the cache.
func (h *handler) messageSetToCache(ctx context.Context, c *message.Message) error {
	if err := h.cache.MessageSet(ctx, c); err != nil {
		return err
	}

	return nil
}

// MessageGet returns message.
func (h *handler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {

	res, err := h.messageGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.messageGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.messageSetToCache(ctx, res)

	return res, nil
}

// MessageDelete deletes the message.
func (h *handler) MessageDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update ai_messages set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. MessageDelete. err: %v", err)
	}

	// update the cache
	_ = h.messageUpdateToCache(ctx, id)

	return nil
}

// MessageGets returns a list of messages.
func (h *handler) MessageGets(ctx context.Context, aicallID uuid.UUID, size uint64, token string, filters map[string]string) ([]*message.Message, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// prepare the query
	q := fmt.Sprintf(`%s
	where
		aicall_id = ?
		and tm_create < ?
	`, messageSelect)

	values := []interface{}{
		aicallID.Bytes(),
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
		return nil, fmt.Errorf("could not query. MessageGets. err: %v", err)
	}
	defer rows.Close()

	res := []*message.Message{}
	for rows.Next() {
		u, err := h.messageGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. MessageGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}
