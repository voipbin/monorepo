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
	"monorepo/bin-chat-manager/models/messagechat"
)

const (
	// select query for messagechat get
	messagechatSelect = `
	select
		id,
		customer_id,

		chat_id,

		source,
		type,
		text,
		medias,

		tm_create,
		tm_update,
		tm_delete
	from
		chat_messagechats
	`
)

// messagechatGetFromRow gets the messagechat from the row.
func (h *handler) messagechatGetFromRow(row *sql.Rows) (*messagechat.Messagechat, error) {
	var source sql.NullString
	var medias sql.NullString

	res := &messagechat.Messagechat{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ChatID,

		&source,
		&res.Type,
		&res.Text,
		&medias,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. messagechatGetFromRow. err: %v", err)
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

// MessagechatCreate creates a new messagechat record
func (h *handler) MessagechatCreate(ctx context.Context, m *messagechat.Messagechat) error {

	q := `insert into chat_messagechats(
		id,
		customer_id,

		chat_id,

		source,
		type,
		text,
		medias,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?,
		?, ?, ?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. MessagechatCreate. err: %v", err)
	}
	defer stmt.Close()

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

		m.ChatID.Bytes(),

		source,
		m.Type,
		m.Text,
		medias,

		m.TMCreate,
		m.TMUpdate,
		m.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. MessagechatCreate. err: %v", err)
	}

	_ = h.messagechatUpdateToCache(ctx, m.ID)

	return nil
}

// messagechatUpdateToCache gets the messagechat from the DB and update the cache.
func (h *handler) messagechatUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.messagechatGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.messagechatSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// messagechatSetToCache sets the given messagechat to the cache
func (h *handler) messagechatSetToCache(ctx context.Context, m *messagechat.Messagechat) error {
	if err := h.cache.MessagechatSet(ctx, m); err != nil {
		return err
	}

	return nil
}

// messagechatGetFromCache returns messagechat from the cache if possible.
func (h *handler) messagechatGetFromCache(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {

	// get from cache
	res, err := h.cache.MessagechatGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// messagechatGetFromDB gets the messagechat info from the db.
func (h *handler) messagechatGetFromDB(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", messagechatSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. messagechatGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. messagechatGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.messagechatGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MessagechatGet returns messagechat.
func (h *handler) MessagechatGet(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error) {

	res, err := h.messagechatGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.messagechatGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.messagechatSetToCache(ctx, res)

	return res, nil
}

// MessagechatGets returns list of message chat.
func (h *handler) MessagechatGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*messagechat.Messagechat, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, messagechatSelect)

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
		return nil, fmt.Errorf("could not query. MessagechatGets. err: %v", err)
	}
	defer rows.Close()

	var res []*messagechat.Messagechat
	for rows.Next() {
		u, err := h.messagechatGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. MessagechatGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil

}

// MessagechatDelete deletes the given messagechat
func (h *handler) MessagechatDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update chat_messagechats set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. MessagechatDelete. err: %v", err)
	}

	// delete cache
	_ = h.messagechatUpdateToCache(ctx, id)

	return nil
}
