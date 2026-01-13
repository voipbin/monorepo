package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-agent-manager/models/agent"
)

const (
	agentTable = "agent_agents"
)

// agentGetFromRow gets the agent from the row.
func (h *handler) agentGetFromRow(row *sql.Rows) (*agent.Agent, error) {
	res := &agent.Agent{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. agentGetFromRow. err: %v", err)
	}

	// Initialize nil slices to empty slices
	if res.TagIDs == nil {
		res.TagIDs = []uuid.UUID{}
	}
	if res.Addresses == nil {
		res.Addresses = []commonaddress.Address{}
	}

	return res, nil
}

// AgentCreate creates new agent record and returns the created agent record.
func (h *handler) AgentCreate(ctx context.Context, a *agent.Agent) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	a.TMCreate = now
	a.TMUpdate = commondatabasehandler.DefaultTimeStamp
	a.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(a)
	if err != nil {
		return fmt.Errorf("could not prepare fields. AgentCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(agentTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AgentCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. AgentCreate. err: %v", err)
	}

	// update the cache
	_ = h.agentUpdateToCache(ctx, a.ID)

	return nil
}

// agentUpdateToCache gets the agent from the DB and update the cache.
func (h *handler) agentUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.agentGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.agentSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// agentSetToCache sets the given agent to the cache
func (h *handler) agentSetToCache(ctx context.Context, u *agent.Agent) error {
	if err := h.cache.AgentSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// agentGetFromCache returns agent from the cache.
func (h *handler) agentGetFromCache(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	// get from cache
	res, err := h.cache.AgentGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// agentGetFromDB returns agent from the DB.
func (h *handler) agentGetFromDB(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	fields := commondatabasehandler.GetDBFields(&agent.Agent{})

	query, args, err := squirrel.
		Select(fields...).
		From(agentTable).
		Where(squirrel.Eq{string(agent.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. agentGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. agentGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. agentGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.agentGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. agentGetFromDB. id: %s", id)
	}

	return res, nil
}

// AgentGet returns agent.
func (h *handler) AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	res, err := h.agentGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.agentGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.agentSetToCache(ctx, res)

	return res, nil
}

// AgentGets returns agents.
func (h *handler) AgentGets(ctx context.Context, size uint64, token string, filters map[agent.Field]any) ([]*agent.Agent, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&agent.Agent{})

	sb := squirrel.
		Select(fields...).
		From(agentTable).
		Where(squirrel.Lt{string(agent.FieldTMCreate): token}).
		OrderBy(string(agent.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. AgentGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. AgentGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AgentGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*agent.Agent{}
	for rows.Next() {
		u, err := h.agentGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. AgentGets. err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. AgentGets. err: %v", err)
	}

	return res, nil
}

// AgentGetByCustomerIDAndAddress returns agent of the given customerID and address.
func (h *handler) AgentGetByCustomerIDAndAddress(ctx context.Context, customerID uuid.UUID, address *commonaddress.Address) (*agent.Agent, error) {
	fields := commondatabasehandler.GetDBFields(&agent.Agent{})

	query, args, err := squirrel.
		Select(fields...).
		From(agentTable).
		Where(squirrel.Eq{string(agent.FieldCustomerID): customerID.Bytes()}).
		Where(squirrel.GtOrEq{string(agent.FieldTMDelete): commondatabasehandler.DefaultTimeStamp}).
		Where("json_contains(addresses, JSON_OBJECT('type', ?, 'target', ?))", address.Type, address.Target).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. AgentGetByCustomerIDAndAddress. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AgentGetByCustomerIDAndAddress. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.agentGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. AgentGetByCustomerIDAndAddress. err: %v", err)
	}

	return res, nil
}

// AgentGetByUsername returns agent of the given username.
func (h *handler) AgentGetByUsername(ctx context.Context, username string) (*agent.Agent, error) {
	filters := map[agent.Field]any{
		agent.FieldDeleted:  false,
		agent.FieldUsername: username,
	}

	tmp, err := h.AgentGets(ctx, 1, h.utilHandler.TimeGetCurTime(), filters)
	if err != nil {
		return nil, err
	}

	if len(tmp) == 0 {
		return nil, ErrNotFound
	}

	return tmp[0], nil
}

// AgentDelete deletes the agent.
func (h *handler) AgentDelete(ctx context.Context, id uuid.UUID) error {
	now := h.utilHandler.TimeGetCurTime()
	fields := map[agent.Field]any{
		agent.FieldTMDelete: now,
		agent.FieldTMUpdate: now,
	}

	if err := h.agentUpdate(ctx, id, fields); err != nil {
		return fmt.Errorf("could not update agent for delete. AgentDelete. err: %v", err)
	}

	return nil
}

// AgentUpdate updates the agent with the given fields.
func (h *handler) AgentUpdate(ctx context.Context, id uuid.UUID, fields map[agent.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[agent.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	return h.agentUpdate(ctx, id, fields)
}

func (h *handler) agentUpdate(ctx context.Context, id uuid.UUID, fields map[agent.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("agentUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(agentTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{"id": id.Bytes()})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("agentUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("agentUpdate: exec failed: %w", err)
	}

	_ = h.agentUpdateToCache(ctx, id)
	return nil
}

// AgentSetBasicInfo sets the agent's basic info.
func (h *handler) AgentSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) error {
	fields := map[agent.Field]any{
		agent.FieldName:       name,
		agent.FieldDetail:     detail,
		agent.FieldRingMethod: ringMethod,
	}

	return h.AgentUpdate(ctx, id, fields)
}

// AgentSetPasswordHash sets the agent password_hash.
func (h *handler) AgentSetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	fields := map[agent.Field]any{
		agent.FieldPasswordHash: passwordHash,
	}

	return h.AgentUpdate(ctx, id, fields)
}

// AgentSetStatus sets the agent status.
func (h *handler) AgentSetStatus(ctx context.Context, id uuid.UUID, status agent.Status) error {
	fields := map[agent.Field]any{
		agent.FieldStatus: status,
	}

	return h.AgentUpdate(ctx, id, fields)
}

// AgentSetPermission sets the agent permission.
func (h *handler) AgentSetPermission(ctx context.Context, id uuid.UUID, permission agent.Permission) error {
	fields := map[agent.Field]any{
		agent.FieldPermission: permission,
	}

	return h.AgentUpdate(ctx, id, fields)
}

// AgentSetTagIDs sets the agent tag_ids.
func (h *handler) AgentSetTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) error {
	fields := map[agent.Field]any{
		agent.FieldTagIDs: tagIDs,
	}

	return h.AgentUpdate(ctx, id, fields)
}

// AgentSetAddresses sets the agent addresses.
func (h *handler) AgentSetAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) error {
	fields := map[agent.Field]any{
		agent.FieldAddresses: addresses,
	}

	return h.AgentUpdate(ctx, id, fields)
}
