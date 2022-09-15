package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
)

const (
	// select query for  messagechatroom  get
	messagechatroomSelect = `
	select
		id,
		customer_id,

		chatroom_id,

		messagechat_id,

		message json,

		tm_create,
		tm_update,
		tm_delete
	from
		 messagechatrooms
	`
)

//  messagechatroomGetFromRow gets the messagechatroom from the row.
func (h *handler) messagechatroomGetFromRow(row *sql.Rows) (*messagechatroom.Messagechatroom, error) {
	var msg sql.NullString

	res := &messagechatroom.Messagechatroom{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ChatroomID,

		&res.MessagechatID,

		&msg,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. messagechatroomGetFromRow. err: %v", err)
	}

	if msg.Valid {
		if err := json.Unmarshal([]byte(msg.String), &res.Message); err != nil {
			return nil, fmt.Errorf("could not unmarshal the data. messagechatroomGetFromRow. err: %v", err)
		}
	} else {
		res.Message = message.Message{}
	}

	return res, nil
}

// MessagechatroomCreate creates a new messagechatroom record
func (h *handler) MessagechatroomCreate(ctx context.Context, m *messagechatroom.Messagechatroom) error {

	q := `insert into messagechatrooms(
		id,
		customer_id,

		chatroom_id,

		messagechat_id,

		message,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?,
		?,
		?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. MessagechatroomCreate. err: %v", err)
	}
	defer stmt.Close()

	msg, err := json.Marshal(m.Message)
	if err != nil {
		return fmt.Errorf("could not marshal actions. MessagechatroomCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		m.ID.Bytes(),
		m.CustomerID.Bytes(),

		m.ChatroomID.Bytes(),

		m.MessagechatID.Bytes(),

		msg,

		m.TMCreate,
		m.TMUpdate,
		m.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. MessagechatroomCreate. err: %v", err)
	}

	_ = h.messagechatroomUpdateToCache(ctx, m.ID)

	return nil
}

// messagechatroomUpdateToCache gets the messagechatroom from the DB and update the cache.
func (h *handler) messagechatroomUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.messagechatroomGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.messagechatroomSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// messagechatroomSetToCache sets the given messagechatroom to the cache
func (h *handler) messagechatroomSetToCache(ctx context.Context, m *messagechatroom.Messagechatroom) error {
	if err := h.cache.MessagechatroomSet(ctx, m); err != nil {
		return err
	}

	return nil
}

// messagechatroomGetFromCache returns messagechatroom from the cache if possible.
func (h *handler) messagechatroomGetFromCache(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {

	// get from cache
	res, err := h.cache.MessagechatroomGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// messagechatroomGetFromDB gets the messagechatroom info from the db.
func (h *handler) messagechatroomGetFromDB(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", messagechatroomSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. messagechatroomGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. messagechatroomGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.messagechatroomGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MessagechatroomGet returns messagechatroom.
func (h *handler) MessagechatroomGet(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {

	res, err := h.messagechatroomGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.messagechatroomGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.messagechatroomSetToCache(ctx, res)

	return res, nil
}

// MessagechatroomGetsByCustomerID returns list of messagechatrooms.
func (h *handler) MessagechatroomGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*messagechatroom.Messagechatroom, error) {

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
	`, messagechatroomSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessagechatroomGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	var res []*messagechatroom.Messagechatroom
	for rows.Next() {
		u, err := h.messagechatroomGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. MessagechatroomGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// MessagechatroomGetsByChatroomID returns list of messagechatrooms of the given chatroom_id.
func (h *handler) MessagechatroomGetsByChatroomID(ctx context.Context, chatroomID uuid.UUID, token string, limit uint64) ([]*messagechatroom.Messagechatroom, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and chatroom_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, messagechatroomSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, chatroomID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessagechatroomGetsByChatroomID. err: %v", err)
	}
	defer rows.Close()

	var res []*messagechatroom.Messagechatroom
	for rows.Next() {
		u, err := h.messagechatroomGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. MessagechatroomGetsByChatroomID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// MessagechatroomGetsByMessagechatID returns list of messagechatrooms of the given messagechat_id.
func (h *handler) MessagechatroomGetsByMessagechatID(ctx context.Context, messagechatID uuid.UUID, token string, limit uint64) ([]*messagechatroom.Messagechatroom, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and messagechat_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, messagechatroomSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, messagechatID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. MessagechatroomGetsByMessagechatID. err: %v", err)
	}
	defer rows.Close()

	var res []*messagechatroom.Messagechatroom
	for rows.Next() {
		u, err := h.messagechatroomGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. MessagechatroomGetsByMessagechatID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// MessagechatroomDelete deletes the given messagechat
func (h *handler) MessagechatroomDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update messagechatrooms set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, GetCurTime(), GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. MessagechatroomDelete. err: %v", err)
	}

	// delete cache
	_ = h.messagechatroomUpdateToCache(ctx, id)

	return nil
}
