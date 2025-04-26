package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/conversation"
)

const (
	conversationsTable  = "conversation_conversations"
	conversationsFields = `
		id,
		customer_id,
		owner_type,
		owner_id,

		account_id,

		name,
		detail,

		type,
		dialog_id,

		self,
		peer,
 
		tm_create,
		tm_update,
		tm_delete
	`
)

// conversationGetFromRow gets the conversation from the row.
func (h *handler) conversationGetFromRow(row *sql.Rows) (*conversation.Conversation, error) {
	var self sql.NullString
	var peer sql.NullString

	res := &conversation.Conversation{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.OwnerType,
		&res.OwnerID,

		&res.AccountID,

		&res.Name,
		&res.Detail,

		&res.Type,
		&res.DialogID,

		&self,
		&peer,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. conversationGetFromRow. err: %v", err)
	}

	if !self.Valid {
		res.Self = commonaddress.Address{}
	} else {
		if err := json.Unmarshal([]byte(self.String), &res.Self); err != nil {
			return nil, fmt.Errorf("could not unmarshal the Source. conversationGetFromRow. err: %v", err)
		}
	}

	if !peer.Valid {
		res.Peer = commonaddress.Address{}
	} else {
		if err := json.Unmarshal([]byte(peer.String), &res.Peer); err != nil {
			return nil, fmt.Errorf("could not unmarshal the Destination. conversationGetFromRow. err: %v", err)
		}
	}

	return res, nil
}

func (h *handler) ConversationCreate(ctx context.Context, cv *conversation.Conversation) error {
	self, err := json.Marshal(cv.Self)
	if err != nil {
		return fmt.Errorf("could not marshal self. ConversationCreate. err: %v", err)
	}

	peer, err := json.Marshal(cv.Peer)
	if err != nil {
		return fmt.Errorf("could not marshal peer. ConversationCreate. err: %v", err)
	}

	now := h.utilHandler.TimeGetCurTime()

	sb := squirrel.
		Insert(conversationsTable).
		Columns(conversationsFields).
		Values(
			cv.ID.Bytes(),
			cv.CustomerID.Bytes(),
			cv.OwnerType,
			cv.OwnerID.Bytes(),
			cv.AccountID.Bytes(),
			cv.Name,
			cv.Detail,
			cv.Type,
			cv.DialogID,
			self,
			peer,
			now,
			commondatabasehandler.DefaultTimeStamp,
			commondatabasehandler.DefaultTimeStamp,
		).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ConversationCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ConversationCreate. err: %v", err)
	}

	_ = h.conversationUpdateToCache(ctx, cv.ID)

	return nil
}

// conversationGetFromDB gets the conversation info from the db.
func (h *handler) conversationGetFromDB(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error) {
	query, args, err := squirrel.
		Select(conversationsFields).
		From(conversationsTable).
		Where(squirrel.Eq{"id": id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. conversationGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conversationGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get conversation. conversationGetFromDB. err: %v", err)
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

func (h *handler) ConversationGetBySelfAndPeer(ctx context.Context, self commonaddress.Address, peer commonaddress.Address) (*conversation.Conversation, error) {
	sb := squirrel.
		Select(conversationsFields).
		From(conversationsTable).
		Where(squirrel.Expr("JSON_UNQUOTE(JSON_EXTRACT(self, '$.type')) = ?", self.Type)).
		Where(squirrel.Expr("JSON_UNQUOTE(JSON_EXTRACT(self, '$.target')) = ?", self.Target)).
		Where(squirrel.Expr("JSON_UNQUOTE(JSON_EXTRACT(peer, '$.type')) = ?", peer.Type)).
		Where(squirrel.Expr("JSON_UNQUOTE(JSON_EXTRACT(peer, '$.target')) = ?", peer.Target)).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ConversationGetBySelfAndPeer. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConversationGetBySelfAndPeer. err: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conversationGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not get conversation. ConversationGetBySelfAndPeer. err: %v", err)
	}

	return res, nil
}

func (h *handler) ConversationGets(ctx context.Context, size uint64, token string, fiedls map[conversation.Field]any) ([]*conversation.Conversation, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	sb := squirrel.
		Select(conversationsFields).
		From(conversationsTable).
		Where(squirrel.Lt{"tm_create": token}).
		OrderBy("tm_create DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	var err error
	sb, err = commondatabasehandler.ApplyFields(sb, fiedls)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ConversationGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ConversationGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
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

// ConversationUpdate updates the conversation info.
func (h *handler) ConversationUpdate(ctx context.Context, id uuid.UUID, fields map[conversation.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields["tm_update"] = h.utilHandler.TimeGetCurTime()

	tmpFields := commondatabasehandler.PrepareUpdateFields(fields)
	q := squirrel.Update(conversationsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{"id": id.Bytes()})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ConversationUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("ConversationUpdate: exec failed: %w", err)
	}

	_ = h.conversationUpdateToCache(ctx, id)
	return nil
}
