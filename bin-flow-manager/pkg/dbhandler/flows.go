package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
)

const (
	// select query for flow get
	flowSelect = `
	select
		id,
		customer_id,
		type,

		name,
		detail,

		actions,

		tm_create,
		tm_update,
		tm_delete
	from
		flow_flows
	`
)

// flowGetFromRow gets the flow from the row.
func (h *handler) flowGetFromRow(row *sql.Rows) (*flow.Flow, error) {
	var actions string

	res := &flow.Flow{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.Type,

		&res.Name,
		&res.Detail,

		&actions,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. flowGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(actions), &res.Actions); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. FlowGet. err: %v", err)
	}
	res.Persist = true

	return res, nil
}

func (h *handler) FlowCreate(ctx context.Context, f *flow.Flow) error {

	q := `insert into flow_flows(
		id,
		customer_id,
		type,

		name,
		detail,

		actions,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. FlowCreate. err: %v", err)
	}
	defer stmt.Close()

	tmpActions, err := json.Marshal(f.Actions)
	if err != nil {
		return fmt.Errorf("could not marshal actions. FlowCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		f.ID.Bytes(),
		f.CustomerID.Bytes(),
		f.Type,

		f.Name,
		f.Detail,

		tmpActions,

		h.util.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. FlowCreate. err: %v", err)
	}

	_ = h.flowUpdateToCache(ctx, f.ID)

	return nil
}

// flowUpdateToCache gets the flow from the DB and update the cache.
func (h *handler) flowUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.flowGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.FlowSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// FlowSetToCache sets the given flow to the cache
func (h *handler) FlowSetToCache(ctx context.Context, f *flow.Flow) error {
	if err := h.cache.FlowSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// flowGetFromCache returns flow from the cache if possible.
func (h *handler) flowGetFromCache(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {

	// get from cache
	res, err := h.cache.FlowGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// flowDeleteCache deletes cache
func (h *handler) flowDeleteCache(ctx context.Context, id uuid.UUID) error {

	// delete from cache
	err := h.cache.FlowDel(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// flowGetFromDB gets the flow info from the db.
func (h *handler) flowGetFromDB(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", flowSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. FlowGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. FlowGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.flowGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// FlowGet returns flow.
func (h *handler) FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {

	res, err := h.flowGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.flowGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.FlowSetToCache(ctx, res)

	return res, nil
}

// FlowGets returns flows.
func (h *handler) FlowGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*flow.Flow, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, flowSelect)

	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id":
			q = fmt.Sprintf("%s and customer_id = ?", q)
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
		return nil, fmt.Errorf("could not query. FlowGets. err: %v", err)
	}
	defer rows.Close()

	res := []*flow.Flow{}
	for rows.Next() {
		u, err := h.flowGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. FlowGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// FlowUpdate updates the most of flow information.
// except permenant info(i.e. id, timestamp, etc)
func (h *handler) FlowUpdate(ctx context.Context, id uuid.UUID, name, detail string, actions []action.Action) error {
	q := `
	update flow_flows set
		name = ?,
		detail = ?,
		actions = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpActions, err := json.Marshal(actions)
	if err != nil {
		return fmt.Errorf("could not marshal actions. FlowUpdate. err: %v", err)
	}

	if _, err := h.db.Exec(q, name, detail, tmpActions, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. FlowUpdate. err: %v", err)
	}

	// set to the cache
	_ = h.flowUpdateToCache(ctx, id)

	return nil
}

// FlowDelete deletes the given flow
func (h *handler) FlowDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update flow_flows set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.util.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. FlowDelete. err: %v", err)
	}

	// delete cache
	_ = h.flowDeleteCache(ctx, id)

	return nil
}

// FlowUpdateActions updates the actions.
func (h *handler) FlowUpdateActions(ctx context.Context, id uuid.UUID, actions []action.Action) error {
	q := `
	update flow_flows set
		actions = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpActions, err := json.Marshal(actions)
	if err != nil {
		return fmt.Errorf("could not marshal actions. FlowUpdateActions. err: %v", err)
	}

	if _, err := h.db.Exec(q, tmpActions, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. FlowUpdateActions. err: %v", err)
	}

	// set to the cache
	_ = h.flowUpdateToCache(ctx, id)

	return nil
}
