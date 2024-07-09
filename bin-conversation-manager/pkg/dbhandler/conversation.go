package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/conversation"
)

const (
	// select query for conversation get
	conversationSelect = `
	select
		id,
		customer_id,
		owner_type,
		owner_id,

		account_id,

		name,
		detail,

		reference_type,
		reference_id,

		source,
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
	var source sql.NullString
	var participants sql.NullString

	res := &conversation.Conversation{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.OwnerType,
		&res.OwnerID,

		&res.AccountID,

		&res.Name,
		&res.Detail,

		&res.ReferenceType,
		&res.ReferenceID,

		&source,
		&participants,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. conversationGetFromRow. err: %v", err)
	}

	if !source.Valid {
		res.Source = &commonaddress.Address{}
	} else {
		if err := json.Unmarshal([]byte(source.String), &res.Source); err != nil {
			return nil, fmt.Errorf("could not unmarshal the Source. conversationGetFromRow. err: %v", err)
		}
	}

	if !participants.Valid {
		res.Participants = []commonaddress.Address{}
	} else {
		if err := json.Unmarshal([]byte(participants.String), &res.Participants); err != nil {
			return nil, fmt.Errorf("could not unmarshal the Participants. conversationGetFromRow. err: %v", err)
		}
	}

	return res, nil
}

// ConversationCreate creates a new conversation record
func (h *handler) ConversationCreate(ctx context.Context, cv *conversation.Conversation) error {

	q := `insert into conversation_conversations(
		id,
		customer_id,
		owner_type,
		owner_id,

		account_id,

		name,
		detail,

		reference_type,
		reference_id,

		source,
		participants,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?,
		?,
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ConversationCreate. err: %v", err)
	}
	defer stmt.Close()

	source, err := json.Marshal(cv.Source)
	if err != nil {
		return fmt.Errorf("could not marshal source. ConversationCreate. err: %v", err)
	}

	participants, err := json.Marshal(cv.Participants)
	if err != nil {
		return fmt.Errorf("could not marshal current_actions. ConversationCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		cv.ID.Bytes(),
		cv.CustomerID.Bytes(),
		cv.OwnerType,
		cv.OwnerID.Bytes(),

		cv.AccountID.Bytes(),

		cv.Name,
		cv.Detail,

		cv.ReferenceType,
		cv.ReferenceID,

		source,
		participants,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
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
		return nil, fmt.Errorf("could not prepare. conversationGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. conversationGetFromDB. err: %v", err)
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
func (h *handler) ConversationGetByReferenceInfo(ctx context.Context, customerID uuid.UUID, ReferenceType conversation.ReferenceType, ReferenceID string) (*conversation.Conversation, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and reference_type = ?
			and reference_id = ?
	`, conversationSelect)

	row, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), ReferenceType, ReferenceID)
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

// ConversationGets returns a list of conversations.
func (h *handler) ConversationGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*conversation.Conversation, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, conversationSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

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

		case "customer_id", "owner_id", "account_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConversationGets. err: %v", err)
	}
	defer rows.Close()

	res := []*conversation.Conversation{}
	for rows.Next() {
		u, err := h.conversationGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ConversationGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ConversationSet returns sets the conversation info
func (h *handler) ConversationSet(ctx context.Context, id uuid.UUID, name string, detail string) error {

	// prepare
	q := `
	update conversation_conversations set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, name, detail, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConversationSet. err: %v", err)
	}

	// update the cache
	_ = h.conversationUpdateToCache(ctx, id)

	return nil
}
