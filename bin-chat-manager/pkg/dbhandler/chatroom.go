package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chatroom"
)

const (
	// select query for chat get
	chatroomSelect = `
	select
		id,
		customer_id,
		agent_id,

		type,
		chat_id,

		owner_id,
		participant_ids,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	from
		chatrooms
	`
)

// chatroomGetFromRow gets the chat from the row.
func (h *handler) chatroomGetFromRow(row *sql.Rows) (*chatroom.Chatroom, error) {
	var participantIDs string

	res := &chatroom.Chatroom{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.AgentID,

		&res.Type,
		&res.ChatID,

		&res.OwnerID,
		&participantIDs,

		&res.Name,
		&res.Detail,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. chatroomGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(participantIDs), &res.ParticipantIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. chatroomGetFromRow. err: %v", err)
	}

	return res, nil
}

// ChatroomCreate creates a new chat record
func (h *handler) ChatroomCreate(ctx context.Context, c *chatroom.Chatroom) error {

	q := `insert into chatrooms(
		id,
		customer_id,
		agent_id,

		type,
		chat_id,

		owner_id,
		participant_ids,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChatroomCreate. err: %v", err)
	}
	defer stmt.Close()

	participantIDs, err := json.Marshal(c.ParticipantIDs)
	if err != nil {
		return fmt.Errorf("could not marshal actions. ChatroomCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),
		c.AgentID.Bytes(),

		c.Type,
		c.ChatID.Bytes(),

		c.OwnerID.Bytes(),
		participantIDs,

		c.Name,
		c.Detail,

		c.TMCreate,
		c.TMUpdate,
		c.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. ChatroomCreate. err: %v", err)
	}

	_ = h.chatroomUpdateToCache(ctx, c.ID)

	return nil
}

// chatroomUpdateToCache gets the chat from the DB and update the cache.
func (h *handler) chatroomUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.chatroomGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.chatroomSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// chatroomSetToCache sets the given chat to the cache
func (h *handler) chatroomSetToCache(ctx context.Context, f *chatroom.Chatroom) error {
	if err := h.cache.ChatroomSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// chatroomGetFromCache returns chat from the cache if possible.
func (h *handler) chatroomGetFromCache(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {

	// get from cache
	res, err := h.cache.ChatroomGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// chatroomGetFromDB gets the chat info from the db.
func (h *handler) chatroomGetFromDB(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", chatroomSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. chatroomGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. chatroomGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.chatroomGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChatroomGet returns chatroom.
func (h *handler) ChatroomGet(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error) {

	res, err := h.chatroomGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.chatroomGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.chatroomSetToCache(ctx, res)

	return res, nil
}

// ChatroomGets returns list of chatrooms.
func (h *handler) ChatroomGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*chatroom.Chatroom, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, chatroomSelect)

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id":
			tmp := uuid.FromStringOrNil(v)
			q = fmt.Sprintf("%s and customer_id = ?", q)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		case "type":
			q = fmt.Sprintf("%s and type = ?", q)
			values = append(values, v)

		case "agent_id":
			tmp := uuid.FromStringOrNil(v)
			q = fmt.Sprintf("%s and agent_id = ?", q)
			values = append(values, tmp.Bytes())

		case "owner_id":
			tmp := uuid.FromStringOrNil(v)
			q = fmt.Sprintf("%s and owner_id = ?", q)
			values = append(values, tmp.Bytes())

		case "chat_id":
			tmp := uuid.FromStringOrNil(v)
			q = fmt.Sprintf("%s and chat_id = ?", q)
			values = append(values, tmp.Bytes())

		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatroomGets. err: %v", err)
	}
	defer rows.Close()

	var res []*chatroom.Chatroom
	for rows.Next() {
		u, err := h.chatroomGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ChatroomGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ChatroomUpdateBasicInfo updates the basic information.
func (h *handler) ChatroomUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	q := `
	update chatrooms set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, name, detail, h.utilHandler.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ChatroomUpdateBasicInfo. err: %v", err)
	}

	// set to the cache
	_ = h.chatroomUpdateToCache(ctx, id)

	return nil
}

// ChatroomDelete deletes the given chat
func (h *handler) ChatroomDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update chatrooms set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()

	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ChatroomDelete. err: %v", err)
	}

	// delete cache
	_ = h.chatroomUpdateToCache(ctx, id)

	return nil
}

// ChatroomAddParticipantID adds the given participant_id to the participant_ids.
func (h *handler) ChatroomAddParticipantID(ctx context.Context, id, participantID uuid.UUID) error {
	// prepare
	q := `
	update chatrooms set
		participant_ids = json_array_append(
			participant_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, participantID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatroomAddParticipantID. err: %v", err)
	}

	// update the cache
	_ = h.chatroomUpdateToCache(ctx, id)

	return nil
}

// ChatroomRemoveParticipantID removes the given participantID from the participant_ids.
func (h *handler) ChatroomRemoveParticipantID(ctx context.Context, id, participantID uuid.UUID) error {
	// prepare
	q := `
	update chatrooms set
		participant_ids = json_remove(
			participant_ids, replace(
				json_search(
					participant_ids,
					'one',
					?
				),
				'"',
				''
			)
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, participantID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatroomRemoveParticipantID. err: %v", err)
	}

	// update the cache
	_ = h.chatroomUpdateToCache(ctx, id)

	return nil
}
