package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-push-manager/models/notification"
)

const (
	kindPrefTable = "push_kind_prefs"
	chatMuteTable = "push_chat_mutes"
)

// idList returns the ids as the binary(16) values of the columns.
func idList(ids []uuid.UUID) [][]byte {
	res := make([][]byte, 0, len(ids))
	for _, id := range ids {
		res = append(res, id.Bytes())
	}

	return res
}

// insertOnce inserts the row. A row that exists already is a success, so the caller can repeat the request:
// when the insert fails, the row is looked up and found means the insert lost a race or was repeated. The
// error of the driver is not parsed, so it works the same on MySQL and on the sqlite of the tests.
func (h *handler) insertOnce(ctx context.Context, name string, insert sq.InsertBuilder, exists sq.SelectBuilder) error {
	query, args, err := insert.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. %s. err: %w", name, err)
	}

	_, errInsert := h.db.ExecContext(ctx, query, args...)
	if errInsert == nil {
		return nil
	}

	found, err := h.rowExists(ctx, name, exists)
	if err != nil {
		return fmt.Errorf("could not execute. %s. err: %w", name, errInsert)
	}
	if found {
		return nil
	}

	return fmt.Errorf("could not execute. %s. err: %w", name, errInsert)
}

// rowExists returns true if the select returns a row.
func (h *handler) rowExists(ctx context.Context, name string, builder sq.SelectBuilder) (bool, error) {
	query, args, err := builder.ToSql()
	if err != nil {
		return false, fmt.Errorf("could not build query. %s. err: %w", name, err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("could not query. %s. err: %w", name, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	found := rows.Next()
	if errRows := rows.Err(); errRows != nil {
		return false, fmt.Errorf("could not iterate the rows. %s. err: %w", name, errRows)
	}

	return found, nil
}

// execDelete deletes the rows that match the condition.
func (h *handler) execDelete(ctx context.Context, name, table string, where sq.Sqlizer) error {
	query, args, err := sq.Delete(table).Where(where).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. %s. err: %w", name, err)
	}

	if _, err = h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute. %s. err: %w", name, err)
	}

	return nil
}

// queryIDs runs the select and returns the first column of every row as an id.
func (h *handler) queryIDs(ctx context.Context, name string, builder sq.SelectBuilder) ([]uuid.UUID, error) {
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. %s. err: %w", name, err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. %s. err: %w", name, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []uuid.UUID{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, fmt.Errorf("could not scan the row. %s. err: %w", name, err)
		}

		id, err := uuid.FromBytes(b)
		if err != nil {
			return nil, fmt.Errorf("could not read the id. %s. err: %w", name, err)
		}
		res = append(res, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate the rows. %s. err: %w", name, err)
	}

	return res, nil
}

// KindPrefDisable switches the kind off for the agent. Switching off a kind that is off succeeds.
func (h *handler) KindPrefDisable(ctx context.Context, customerID, agentID uuid.UUID, kind notification.Kind) error {
	insert := sq.Insert(kindPrefTable).
		Columns("agent_id", "kind", "customer_id", "tm_create").
		Values(agentID.Bytes(), string(kind), customerID.Bytes(), h.utilHandler.TimeNow())
	exists := sq.Select("1").From(kindPrefTable).
		Where(sq.Eq{"agent_id": agentID.Bytes(), "kind": string(kind)})

	return h.insertOnce(ctx, "KindPrefDisable", insert, exists)
}

// KindPrefEnable switches the kind on for the agent. Switching on a kind that is on succeeds.
func (h *handler) KindPrefEnable(ctx context.Context, customerID, agentID uuid.UUID, kind notification.Kind) error {
	return h.execDelete(ctx, "KindPrefEnable", kindPrefTable, sq.Eq{
		"customer_id": customerID.Bytes(),
		"agent_id":    agentID.Bytes(),
		"kind":        string(kind),
	})
}

// KindPrefListDisabledByAgent returns the kinds the agent switched off, ordered by the kind.
func (h *handler) KindPrefListDisabledByAgent(ctx context.Context, customerID, agentID uuid.UUID) ([]notification.Kind, error) {
	query, args, err := sq.Select("kind").From(kindPrefTable).
		Where(sq.Eq{"customer_id": customerID.Bytes(), "agent_id": agentID.Bytes()}).
		OrderBy("kind").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. KindPrefListDisabledByAgent. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. KindPrefListDisabledByAgent. err: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []notification.Kind{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("could not scan the row. KindPrefListDisabledByAgent. err: %w", err)
		}
		res = append(res, notification.Kind(k))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate the rows. KindPrefListDisabledByAgent. err: %w", err)
	}

	return res, nil
}

// KindPrefListDisabledByAgents returns the agents of the list that switched the kind off.
func (h *handler) KindPrefListDisabledByAgents(ctx context.Context, customerID uuid.UUID, agentIDs []uuid.UUID, kind notification.Kind) ([]uuid.UUID, error) {
	if len(agentIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	builder := sq.Select("agent_id").From(kindPrefTable).
		Where(sq.Eq{"customer_id": customerID.Bytes(), "kind": string(kind), "agent_id": idList(agentIDs)}).
		OrderBy("agent_id")

	return h.queryIDs(ctx, "KindPrefListDisabledByAgents", builder)
}

// KindPrefDeleteByAgentID deletes the kind settings of the agent.
func (h *handler) KindPrefDeleteByAgentID(ctx context.Context, agentID uuid.UUID) error {
	return h.execDelete(ctx, "KindPrefDeleteByAgentID", kindPrefTable, sq.Eq{"agent_id": agentID.Bytes()})
}

// KindPrefDeleteByCustomerID deletes the kind settings of the customer.
func (h *handler) KindPrefDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error {
	return h.execDelete(ctx, "KindPrefDeleteByCustomerID", kindPrefTable, sq.Eq{"customer_id": customerID.Bytes()})
}

// MuteCreate mutes the chat for the agent. Muting a muted chat succeeds.
func (h *handler) MuteCreate(ctx context.Context, customerID, agentID, chatID uuid.UUID) error {
	insert := sq.Insert(chatMuteTable).
		Columns("agent_id", "chat_id", "customer_id", "tm_create").
		Values(agentID.Bytes(), chatID.Bytes(), customerID.Bytes(), h.utilHandler.TimeNow())
	exists := sq.Select("1").From(chatMuteTable).
		Where(sq.Eq{"agent_id": agentID.Bytes(), "chat_id": chatID.Bytes()})

	return h.insertOnce(ctx, "MuteCreate", insert, exists)
}

// MuteDelete unmutes the chat for the agent. Unmuting a chat that is not muted succeeds.
func (h *handler) MuteDelete(ctx context.Context, customerID, agentID, chatID uuid.UUID) error {
	return h.execDelete(ctx, "MuteDelete", chatMuteTable, sq.Eq{
		"customer_id": customerID.Bytes(),
		"agent_id":    agentID.Bytes(),
		"chat_id":     chatID.Bytes(),
	})
}

// MuteListByAgent returns the chats the agent muted, the oldest mute first.
func (h *handler) MuteListByAgent(ctx context.Context, customerID, agentID uuid.UUID) ([]uuid.UUID, error) {
	builder := sq.Select("chat_id").From(chatMuteTable).
		Where(sq.Eq{"customer_id": customerID.Bytes(), "agent_id": agentID.Bytes()}).
		OrderBy("tm_create", "chat_id")

	return h.queryIDs(ctx, "MuteListByAgent", builder)
}

// MuteListMutedByAgents returns the agents of the list that muted the chat.
func (h *handler) MuteListMutedByAgents(ctx context.Context, customerID uuid.UUID, agentIDs []uuid.UUID, chatID uuid.UUID) ([]uuid.UUID, error) {
	if len(agentIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	builder := sq.Select("agent_id").From(chatMuteTable).
		Where(sq.Eq{"customer_id": customerID.Bytes(), "chat_id": chatID.Bytes(), "agent_id": idList(agentIDs)}).
		OrderBy("agent_id")

	return h.queryIDs(ctx, "MuteListMutedByAgents", builder)
}

// MuteDeleteByAgentID deletes the muted chats of the agent.
func (h *handler) MuteDeleteByAgentID(ctx context.Context, agentID uuid.UUID) error {
	return h.execDelete(ctx, "MuteDeleteByAgentID", chatMuteTable, sq.Eq{"agent_id": agentID.Bytes()})
}

// MuteDeleteByCustomerID deletes the muted chats of the customer.
func (h *handler) MuteDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error {
	return h.execDelete(ctx, "MuteDeleteByCustomerID", chatMuteTable, sq.Eq{"customer_id": customerID.Bytes()})
}
