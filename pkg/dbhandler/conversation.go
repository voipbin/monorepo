package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

const (
	// select query for conversation get
	conversationSelect = `
	select
		id,
		customer_id,

		name,
		detail,

		reference_type,
		reference_id,

		participants,

		tm_create,
		tm_update,
		tm_delete
	from
		conversation_conversations
	`
)

// conversationGetFromRow gets the conversation from the row.
func (h *handler) conversationGetFromRow(row *sql.Rows) (*conversation.Conversation, error) {
	participants := ""

	res := &conversation.Conversation{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,

		&res.ReferenceType,
		&res.ReferenceID,

		&participants,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. conversationGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(participants), &res.Participants); err != nil {
		return nil, fmt.Errorf("could not unmarshal the Participants. conversationGetFromRow. err: %v", err)
	}

	return res, nil
}

// ConversationCreate creates a new conversation record
func (h *handler) ConversationCreate(ctx context.Context, cv *conversation.Conversation) error {

	q := `insert into conversation_conversations(
		id,
		customer_id,

		name,
		detail,

		reference_type,
		reference_id,

		participants,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?,
		?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ConversationCreate. err: %v", err)
	}
	defer stmt.Close()

	participants, err := json.Marshal(cv.Participants)
	if err != nil {
		return fmt.Errorf("could not marshal current_actions. ConversationCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		cv.ID.Bytes(),
		cv.CustomerID.Bytes(),

		cv.Name,
		cv.Detail,

		cv.ReferenceType,
		cv.ReferenceID,

		participants,

		cv.TMCreate,
		cv.TMUpdate,
		cv.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. ConversationCreate. err: %v", err)
	}

	_ = h.conversationUpdateToCache(ctx, cv.ID)

	return nil
}

// conversationGetFromDB gets the conversation info from the db.
func (h *handler) conversationGetFromDB(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", conversationSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. activeflowGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. activeflowGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conversationGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// conversationUpdateToCache gets the conversation from the DB and update the cache.
func (h *handler) conversationUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.conversationGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.conversationSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// conversationSetToCache sets the given conversation to the cache
func (h *handler) conversationSetToCache(ctx context.Context, flow *conversation.Conversation) error {
	if err := h.cache.ConversationSet(ctx, flow); err != nil {
		return err
	}

	return nil
}

// conversationGetFromCache returns conversation from the cache.
func (h *handler) conversationGetFromCache(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error) {

	// get from cache
	res, err := h.cache.ConversationGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ConversationGetByReferenceInfo returns conversation by the reference.
func (h *handler) ConversationGetByReferenceInfo(ctx context.Context, ReferenceType conversation.ReferenceType, ReferenceID string) (*conversation.Conversation, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and reference_type = ?
			and reference_id = ?
	`, conversationSelect)

	row, err := h.db.Query(q, DefaultTimeStamp, ReferenceType, ReferenceID)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConversationGetByReferenceInfo. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conversationGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get conversation. ConversationGetByReferenceInfo. err: %v", err)
	}

	return res, nil
}

// ConversationGet returns conversation.
func (h *handler) ConversationGet(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error) {

	res, err := h.conversationGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.conversationGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.conversationSetToCache(ctx, res)

	return res, nil
}

// ConversationGetsByCustomerID returns list of conversation.
func (h *handler) ConversationGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*conversation.Conversation, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, conversationSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConversationGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	var res []*conversation.Conversation
	for rows.Next() {
		u, err := h.conversationGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. ConversationGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}
