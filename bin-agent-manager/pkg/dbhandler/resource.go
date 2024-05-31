package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"monorepo/bin-agent-manager/models/resource"
	"strconv"

	"github.com/gofrs/uuid"
)

const (
	// select query for agent get
	resourceSelect = `
	select
		id,
		customer_id,
		agent_id,
	
		reference_type,
		reference_id,
	
		data,
	
		tm_create,
		tm_update,
		tm_delete
	from
		agent_resources
	`
)

// resourceGetFromRow gets the agent from the row.
func (h *handler) resourceGetFromRow(row *sql.Rows) (*resource.Resource, error) {

	res := &resource.Resource{}

	tmp := []byte{}

	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.AgentID,

		&res.ReferenceType,
		&res.ReferenceID,

		// &res.Data,
		&tmp,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. resourceGetFromRow. err: %v", err)
	}

	if errJson := json.Unmarshal(tmp, res.Data); errJson != nil {
		return nil, errJson
	}

	return res, nil
}

// ResourceCreate creates new agent record and returns the created agent record.
func (h *handler) ResourceCreate(ctx context.Context, a *resource.Resource) error {
	q := `insert into agent_resources(
		id,
		customer_id,
		agent_id,

		reference_type,
		reference_id,

		data,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?,
		?, ?, ?
		)
	`

	tmpData, err := json.Marshal(a.Data)
	if err != nil {
		return fmt.Errorf("could not marshal the data. err: %v", err)
	}

	_, err = h.db.Exec(q,
		a.ID.Bytes(),
		a.CustomerID.Bytes(),
		a.AgentID.Bytes(),

		a.ReferenceType,
		a.ReferenceID.Bytes(),

		// a.Data,
		tmpData,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ResourceCreate. err: %v", err)
	}

	// update the cache
	_ = h.ResourceUpdateToCache(ctx, a.ID)

	return nil
}

// ResourceUpdateToCache gets the agent from the DB and update the cache.
func (h *handler) ResourceUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.resourceGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.resourceSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// resourceSetToCache sets the given agent to the cache
func (h *handler) resourceSetToCache(ctx context.Context, u *resource.Resource) error {
	if err := h.cache.ResourceSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// resourceGetFromCache returns agent from the cache.
func (h *handler) resourceGetFromCache(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {

	// get from cache
	res, err := h.cache.ResourceGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// resourceGetFromDB returns agent from the DB.
func (h *handler) resourceGetFromDB(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", resourceSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. resourceGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.resourceGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. resourceGetFromDB. err: %v", err)
	}

	return res, nil
}

// ResourceGet returns agent.
func (h *handler) ResourceGet(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {
	res, err := h.resourceGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.resourceGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.resourceSetToCache(ctx, res)

	return res, nil
}

// ResourceGets returns resources.
func (h *handler) ResourceGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*resource.Resource, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, resourceSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "agent_id", "reference_id":
			q = fmt.Sprintf("%s and %s = ?", k, q)
			tmp := uuid.FromStringOrNil(v)
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
		return nil, fmt.Errorf("could not query. ResourceGets. err: %v", err)
	}
	defer rows.Close()

	res := []*resource.Resource{}
	for rows.Next() {
		u, err := h.resourceGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. ResourceGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ResourceDelete deletes the agent.
func (h *handler) ResourceDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		agent_resources
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ResourceDelete. err: %v", err)
	}

	// update the cache
	_ = h.ResourceUpdateToCache(ctx, id)

	return nil
}

// ResourceSetData sets the agent's data.
func (h *handler) ResourceSetData(ctx context.Context, id uuid.UUID, data []byte) error {
	// prepare
	q := `
	update
		agent_resources
	set
		data = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, data, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ResourceSetData. err: %v", err)
	}

	// update the cache
	_ = h.ResourceUpdateToCache(ctx, id)

	return nil
}
