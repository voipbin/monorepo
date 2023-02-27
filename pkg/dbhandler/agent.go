package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

const (
	// select query for agent get
	agentSelect = `
	select
		id,
		customer_id,
		username,
		password_hash,

		name,
		detail,

		ring_method,

		status,
		permission,
		tag_ids,
		addresses,

		tm_create,
		tm_update,
		tm_delete
	from
		agents
	`
)

// agentGetFromRow gets the agent from the row.
func (h *handler) agentGetFromRow(row *sql.Rows) (*agent.Agent, error) {

	tagIDs := ""
	addresses := ""

	res := &agent.Agent{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.Username,
		&res.PasswordHash,

		&res.Name,
		&res.Detail,

		&res.RingMethod,

		&res.Status,
		&res.Permission,
		&tagIDs,
		&addresses,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. agentGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(tagIDs), &res.TagIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the tag_ids. agentGetFromRow. err: %v", err)
	}
	if res.TagIDs == nil {
		res.TagIDs = []uuid.UUID{}
	}

	if err := json.Unmarshal([]byte(addresses), &res.Addresses); err != nil {
		return nil, fmt.Errorf("could not unmarshal the endpoints. agentGetFromRow. err: %v", err)
	}
	if res.Addresses == nil {
		res.Addresses = []commonaddress.Address{}
	}

	return res, nil
}

// AgentCreate creates new agent record and returns the created agent record.
func (h *handler) AgentCreate(ctx context.Context, a *agent.Agent) error {
	q := `insert into agents(
		id,
		customer_id,
		username,
		password_hash,

		name,
		detail,

		ring_method,

		status,
		permission,
		tag_ids,
		addresses,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?,
		?, ?,
		?,
		?, ?, ?, ?,
		?, ?, ?
		)
	`

	tagIDs, err := json.Marshal(a.TagIDs)
	if err != nil {
		return fmt.Errorf("could not marshal the tag_ids. err: %v", err)
	}
	addresses, err := json.Marshal(a.Addresses)
	if err != nil {
		return fmt.Errorf("could not marshal the addresses. err: %v", err)
	}

	_, err = h.db.Exec(q,
		a.ID.Bytes(),
		a.CustomerID.Bytes(),
		a.Username,
		a.PasswordHash,

		a.Name,
		a.Detail,

		a.RingMethod,

		a.Status,
		a.Permission,
		tagIDs,
		addresses,

		h.utilHandler.GetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. AgentCreate. err: %v", err)
	}

	// update the cache
	_ = h.AgentUpdateToCache(ctx, a.ID)

	return nil
}

// AgentUpdateToCache gets the agent from the DB and update the cache.
func (h *handler) AgentUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.AgentGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.AgentSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// AgentSetToCache sets the given agent to the cache
func (h *handler) AgentSetToCache(ctx context.Context, u *agent.Agent) error {
	if err := h.cache.AgentSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// AgentGetFromCache returns agent from the cache.
func (h *handler) AgentGetFromCache(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {

	// get from cache
	res, err := h.cache.AgentGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AgentGetFromDB returns agent from the DB.
func (h *handler) AgentGetFromDB(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", agentSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. GetFromCache. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.agentGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. GetFromCache. err: %v", err)
	}

	return res, nil
}

// AgentGet returns agent.
func (h *handler) AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	res, err := h.AgentGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.AgentGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.AgentSetToCache(ctx, res)

	return res, nil
}

// AgentGets returns agents.
func (h *handler) AgentGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*agent.Agent, error) {
	// prepare
	q := fmt.Sprintf("%s where customer_id = ? and tm_create < ? order by tm_create desc limit ?", agentSelect)

	rows, err := h.db.Query(q, customerID.Bytes(), token, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. AgentGets. err: %v", err)
	}
	defer rows.Close()

	var res []*agent.Agent
	for rows.Next() {
		u, err := h.agentGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. AgentGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// AgentGetByUsername returns agent of the given username.
func (h *handler) AgentGetByUsername(ctx context.Context, customerID uuid.UUID, username string) (*agent.Agent, error) {
	// prepare
	q := fmt.Sprintf("%s where customer_id = ? and username = ?", agentSelect)

	row, err := h.db.Query(q, customerID.Bytes(), username)
	if err != nil {
		return nil, fmt.Errorf("could not query. AgentGetByUsername. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.agentGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. AgentGetByUsername. err: %v", err)
	}

	return res, nil
}

// AgentDelete deletes the agent.
func (h *handler) AgentDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		agents
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.GetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AgentDelete. err: %v", err)
	}

	// update the cache
	_ = h.AgentUpdateToCache(ctx, id)

	return nil
}

// AgentSetBasicInfo sets the agent's basic info.
func (h *handler) AgentSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) error {
	// prepare
	q := `
	update
		agents
	set
		name = ?,
		detail = ?,
		ring_method = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, name, detail, ringMethod, h.utilHandler.GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AgentSetBasicInfo. err: %v", err)
	}

	// update the cache
	_ = h.AgentUpdateToCache(ctx, id)

	return nil
}

// AgentSetPasswordHash sets the agent password_hash.
func (h *handler) AgentSetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	// prepare
	q := `
	update
		agents
	set
		password_hash = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, passwordHash, h.utilHandler.GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AgentSetPasswordHash. err: %v", err)
	}

	// update the cache
	_ = h.AgentUpdateToCache(ctx, id)

	return nil
}

// AgentSetStatus sets the agent status.
func (h *handler) AgentSetStatus(ctx context.Context, id uuid.UUID, status agent.Status) error {
	// prepare
	q := `
	update
		agents
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, status, h.utilHandler.GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AgentSetStatus. err: %v", err)
	}

	// update the cache
	_ = h.AgentUpdateToCache(ctx, id)

	return nil
}

// AgentSetPermission sets the agent permission.
func (h *handler) AgentSetPermission(ctx context.Context, id uuid.UUID, permission agent.Permission) error {
	// prepare
	q := `
	update
		agents
	set
		permission = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, permission, h.utilHandler.GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AgentSetPermission. err: %v", err)
	}

	// update the cache
	_ = h.AgentUpdateToCache(ctx, id)

	return nil
}

// AgentSetTagIDs sets the agent tag_ids.
func (h *handler) AgentSetTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) error {
	// prepare
	q := `
	update
		agents
	set
		tag_ids = ?,
		tm_update = ?
	where
		id = ?
	`

	t, err := json.Marshal(tagIDs)
	if err != nil {
		return fmt.Errorf("could not marshal the tag_ids. err: %v", err)
	}

	_, err = h.db.Exec(q, t, h.utilHandler.GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AgentSetPermission. err: %v", err)
	}

	// update the cache
	_ = h.AgentUpdateToCache(ctx, id)

	return nil
}

// AgentSetAddresses sets the agent addresses.
func (h *handler) AgentSetAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) error {
	// prepare
	q := `
	update
		agents
	set
		addresses = ?,
		tm_update = ?
	where
		id = ?
	`

	t, err := json.Marshal(addresses)
	if err != nil {
		return fmt.Errorf("could not marshal the addresses. err: %v", err)
	}

	_, err = h.db.Exec(q, t, h.utilHandler.GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AgentSetAddresses. err: %v", err)
	}

	// update the cache
	_ = h.AgentUpdateToCache(ctx, id)

	return nil
}
