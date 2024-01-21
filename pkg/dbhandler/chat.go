package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
)

const (
	// select query for chat get
	chatSelect = `
	select
		id,
		customer_id,

		type,

		owner_id,
		participant_ids,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	from
		chats
	`
)

// chatGetFromRow gets the chat from the row.
func (h *handler) chatGetFromRow(row *sql.Rows) (*chat.Chat, error) {
	var participantIDs string

	res := &chat.Chat{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Type,

		&res.OwnerID,
		&participantIDs,

		&res.Name,
		&res.Detail,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. chatGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(participantIDs), &res.ParticipantIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. chatGetFromRow. err: %v", err)
	}

	return res, nil
}

// ChatCreate creates a new chat record
func (h *handler) ChatCreate(ctx context.Context, c *chat.Chat) error {

	q := `insert into chats(
		id,
		customer_id,

		type,

		owner_id,
		participant_ids,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?,
		?, ?,
		?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChatCreate. err: %v", err)
	}
	defer stmt.Close()

	participantIDs, err := json.Marshal(c.ParticipantIDs)
	if err != nil {
		return fmt.Errorf("could not marshal actions. ChatCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),

		c.Type,

		c.OwnerID.Bytes(),
		participantIDs,

		c.Name,
		c.Detail,

		c.TMCreate,
		c.TMUpdate,
		c.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. ChatCreate. err: %v", err)
	}

	_ = h.chatUpdateToCache(ctx, c.ID)

	return nil
}

// chatUpdateToCache gets the chat from the DB and update the cache.
func (h *handler) chatUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.chatGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.chatSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// chatSetToCache sets the given chat to the cache
func (h *handler) chatSetToCache(ctx context.Context, c *chat.Chat) error {
	if err := h.cache.ChatSet(ctx, c); err != nil {
		return err
	}

	return nil
}

// chatGetFromCache returns chat from the cache if possible.
func (h *handler) chatGetFromCache(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {

	// get from cache
	res, err := h.cache.ChatGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// chatGetFromDB gets the chat info from the db.
func (h *handler) chatGetFromDB(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", chatSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. chatGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. chatGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.chatGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ChatGet returns chat.
func (h *handler) ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {

	res, err := h.chatGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.chatGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.chatSetToCache(ctx, res)

	return res, nil
}

// ChatGetByTypeAndParticipantsID returns a chat of the given customerID and chatType and participant ids.
func (h *handler) ChatGetByTypeAndParticipantsID(ctx context.Context, customerID uuid.UUID, chatType chat.Type, participantIDs []uuid.UUID) (*chat.Chat, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and type = ?
			and participant_ids = ?
		order by
			tm_create desc, id desc
		limit 1
	`, chatSelect)

	tmp, err := json.Marshal(participantIDs)
	if err != nil {
		return nil, fmt.Errorf("could not marshal actions. ChatGetByTypeAndParticipantsID. err: %v", err)
	}

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), chatType, tmp)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatGetByTypeAndParticipantsID. err: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.chatGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ChatGetByTypeAndParticipantsID. err: %v", err)
	}

	return res, nil
}

// ChatGetsByCustomerID returns list of chats.
func (h *handler) ChatGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, size uint64) ([]*chat.Chat, error) {

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
	`, chatSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), token, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	var res []*chat.Chat
	for rows.Next() {
		u, err := h.chatGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ChatGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ChatGetsByType returns list of chats of the given customerID and chatType.
func (h *handler) ChatGetsByType(ctx context.Context, customerID uuid.UUID, chatType chat.Type, token string, size uint64) ([]*chat.Chat, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and type = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, chatSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), chatType, token, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatGetsByType. err: %v", err)
	}
	defer rows.Close()

	var res []*chat.Chat
	for rows.Next() {
		u, err := h.chatGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ChatGetsByType. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ChatGetsByType returns list of chats of the given customerID and chatType.
func (h *handler) ChatGetsByTypeAndOnwerID(ctx context.Context, customerID uuid.UUID, chatType chat.Type, ownerID uuid.UUID, token string, limit uint64) ([]*chat.Chat, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and type = ?
			and owner_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, chatSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), ownerID.Bytes(), chatType, token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChatGetsByType. err: %v", err)
	}
	defer rows.Close()

	var res []*chat.Chat
	for rows.Next() {
		u, err := h.chatGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ChatGetsByType. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ChatUpdateBasicInfo updates the basic information.
func (h *handler) ChatUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	q := `
	update chats set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, name, detail, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ChatUpdateBasicInfo. err: %v", err)
	}

	// set to the cache
	_ = h.chatUpdateToCache(ctx, id)

	return nil
}

// ChatDelete deletes the given chat
func (h *handler) ChatDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update chats set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, GetCurTime(), GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ChatDelete. err: %v", err)
	}

	// delete cache
	_ = h.chatUpdateToCache(ctx, id)

	return nil
}

// ChatUpdateOwnerID updates the chat's owner_id.
func (h *handler) ChatUpdateOwnerID(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	q := `
	update chats set
		owner_id = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, ownerID.Bytes(), GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ChatUpdateOwnerID. err: %v", err)
	}

	// set to the cache
	_ = h.chatUpdateToCache(ctx, id)

	return nil
}

// ChatAddParticipantID adds the given participant_id to the participant_ids.
func (h *handler) ChatAddParticipantID(ctx context.Context, id, participantID uuid.UUID) error {
	// prepare
	q := `
	update chats set
		participant_ids = json_array_append(
			participant_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, participantID.String(), GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatAddParticipantID. err: %v", err)
	}

	// update the cache
	_ = h.chatUpdateToCache(ctx, id)

	return nil
}

// ChatRemoveParticipantID removes the given participantID from the participant_ids.
func (h *handler) ChatRemoveParticipantID(ctx context.Context, id, participantID uuid.UUID) error {
	// prepare
	q := `
	update chats set
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

	_, err := h.db.Exec(q, participantID.String(), GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatRemoveParticipantID. err: %v", err)
	}

	// update the cache
	_ = h.chatUpdateToCache(ctx, id)

	return nil
}
