package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechatroom"
)

const (
	// select query for  messagechatroom  get
	messagechatroomSelect = `
	select
		id,
		customer_id,
		owner_type,
		owner_id,

		chatroom_id,
		messagechat_id,

		source,
		type,
		text,
		medias,

		tm_create,
		tm_update,
		tm_delete
	from
		chat_messagechatrooms
	`
)

// messagechatroomGetFromRow gets the messagechatroom from the row.
func (h *handler) messagechatroomGetFromRow(row *sql.Rows) (*messagechatroom.Messagechatroom, error) {
	var source sql.NullString
	var medias sql.NullString

	res := &messagechatroom.Messagechatroom{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.OwnerType,
		&res.OwnerID,

		&res.ChatroomID,
		&res.MessagechatID,

		&source,
		&res.Type,
		&res.Text,
		&medias,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. messagechatroomGetFromRow. err: %v", err)
	}

	if source.Valid {
		if err := json.Unmarshal([]byte(source.String), &res.Source); err != nil {
			return nil, fmt.Errorf("could not unmarshal the data. messagechatGetFromRow. err: %v", err)
		}
	} else {
		res.Source = &commonaddress.Address{}
	}

	if medias.Valid {
		if err := json.Unmarshal([]byte(medias.String), &res.Medias); err != nil {
			return nil, fmt.Errorf("could not unmarshal the data. messagechatGetFromRow. err: %v", err)
		}
	} else {
		res.Medias = []media.Media{}
	}
	return res, nil
}

// MessagechatroomCreate creates a new messagechatroom record
func (h *handler) MessagechatroomCreate(ctx context.Context, m *messagechatroom.Messagechatroom) error {

	q := `insert into chat_messagechatrooms(
		id,
		customer_id,
		owner_type,
		owner_id,

		chatroom_id,
		messagechat_id,

		source,
		type,
		text,
		medias,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?,
		?, ?,
		?, ?, ?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. MessagechatroomCreate. err: %v", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	source, err := json.Marshal(m.Source)
	if err != nil {
		return fmt.Errorf("could not marshal source. MessagechatCreate. err: %v", err)
	}

	medias, err := json.Marshal(m.Medias)
	if err != nil {
		return fmt.Errorf("could not marshal medias. MessagechatCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		m.ID.Bytes(),
		m.CustomerID.Bytes(),
		m.OwnerType,
		m.OwnerID.Bytes(),

		m.ChatroomID.Bytes(),
		m.MessagechatID.Bytes(),

		source,
		m.Type,
		m.Text,
		medias,

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
	defer func() {
		_ = stmt.Close()
	}()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. messagechatroomGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

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

// MessagechatroomGets returns list of messagechatrooms.
func (h *handler) MessagechatroomGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*messagechatroom.Messagechatroom, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, messagechatroomSelect)

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "owner_id", "chatroom_id", "messagechat_id":
			tmp := uuid.FromStringOrNil(v)
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)

		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)

	if err != nil {
		return nil, fmt.Errorf("could not query. MessagechatroomGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var res []*messagechatroom.Messagechatroom
	for rows.Next() {
		u, err := h.messagechatroomGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. MessagechatroomGets. err: %v", err)
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
	defer func() {
		_ = rows.Close()
	}()

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
	defer func() {
		_ = rows.Close()
	}()

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
	update chat_messagechatrooms set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. MessagechatroomDelete. err: %v", err)
	}

	// delete cache
	_ = h.messagechatroomUpdateToCache(ctx, id)

	return nil
}
