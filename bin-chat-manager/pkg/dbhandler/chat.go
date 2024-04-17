package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chat"
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

	tmpParticipantIDs := sortUUIDs(c.ParticipantIDs)
	participantIDs, err := json.Marshal(tmpParticipantIDs)
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

// ChatGets returns list of chats.
func (h *handler) ChatGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*chat.Chat, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			tm_create < ?
		`, chatSelect)

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

		case "owner_id":
			tmp := uuid.FromStringOrNil(v)
			q = fmt.Sprintf("%s and owner_id = ?", q)
			values = append(values, tmp.Bytes())

		case "participant_ids":
			tmp := h.chatFilterParseParticipantIDs(v)
			if tmp == "" {
				// has no participant ids
				continue
			}
			values = append(values, tmp)

			q = fmt.Sprintf("%s and participant_ids = json_array(?)", q)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
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

func (h *handler) chatFilterParseParticipantIDs(participantIDs string) string {
	if participantIDs == "" {
		return ""
	}

	ids := strings.Split(participantIDs, ",")
	sort.Strings(ids)

	res := ""
	for i, id := range ids {
		if i == 0 {
			res = fmt.Sprintf(`"%s"`, id)
		} else {
			res = fmt.Sprintf(`%s,"%s"`, res, id)
		}
	}
	res = fmt.Sprintf(`[%s]`, res)

	return res
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

	if _, err := h.db.Exec(q, name, detail, h.utilHandler.TimeGetCurTime(), id.Bytes()); err != nil {
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

	ts := h.utilHandler.TimeGetCurTime()

	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
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

	if _, err := h.db.Exec(q, ownerID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ChatUpdateOwnerID. err: %v", err)
	}

	// set to the cache
	_ = h.chatUpdateToCache(ctx, id)

	return nil
}

// ChatUpdateParticipantID updates the given participant_id to the participant_ids.
func (h *handler) ChatUpdateParticipantID(ctx context.Context, id uuid.UUID, participantIDs []uuid.UUID) error {
	// prepare
	q := `
	update chats set
		participant_ids = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpParticipantIDs := sortUUIDs(participantIDs)
	tmp, err := json.Marshal(tmpParticipantIDs)
	if err != nil {
		return fmt.Errorf("could not marshal actions. ChatUpdateParticipantID. err: %v", err)
	}

	_, err = h.db.Exec(q, tmp, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ChatUpdateParticipantID. err: %v", err)
	}

	// update the cache
	_ = h.chatUpdateToCache(ctx, id)

	return nil
}
