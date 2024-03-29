package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

const (
	// select query for flow get
	activeflowSelect = `
	select
		id,
		customer_id,

		status,
		flow_id,

		reference_type,
		reference_id,

		stack_map,

		current_stack_id,
		current_action,

		forward_stack_id,
		forward_action_id,

		execute_count,
		executed_actions,

		tm_create,
		tm_update,
		tm_delete
	from
		activeflows
	`
)

// activeflowGetFromRow gets the activeflow from the row.
func (h *handler) activeflowGetFromRow(row *sql.Rows) (*activeflow.Activeflow, error) {
	var currentAction string
	var stackMap string
	var executedActions string

	res := &activeflow.Activeflow{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Status,
		&res.FlowID,

		&res.ReferenceType,
		&res.ReferenceID,

		&stackMap,

		&res.CurrentStackID,
		&currentAction,

		&res.ForwardStackID,
		&res.ForwardActionID,

		&res.ExecuteCount,
		&executedActions,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. activeflowGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(currentAction), &res.CurrentAction); err != nil {
		return nil, fmt.Errorf("could not unmarshal the CurrentAction. activeflowGetFromRow. err: %v", err)
	}
	if err := json.Unmarshal([]byte(executedActions), &res.ExecutedActions); err != nil {
		return nil, fmt.Errorf("could not unmarshal the ExecutedActions. activeflowGetFromRow. err: %v", err)
	}
	if err := json.Unmarshal([]byte(stackMap), &res.StackMap); err != nil {
		return nil, fmt.Errorf("could not unmarshal the StackMap. activeflowGetFromRow. err: %v", err)
	}

	return res, nil
}

// ActiveflowCreate creates a new activeflow record
func (h *handler) ActiveflowCreate(ctx context.Context, f *activeflow.Activeflow) error {

	q := `insert into activeflows(
		id,
		customer_id,

		status,
		flow_id,

		reference_type,
		reference_id,

		stack_map,

		current_stack_id,
		current_action,

		forward_stack_id,
		forward_action_id,

		execute_count,
		executed_actions,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?,
		?,
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ActiveflowCreate. err: %v", err)
	}
	defer stmt.Close()

	tmpCurrentAction, err := json.Marshal(f.CurrentAction)
	if err != nil {
		return fmt.Errorf("could not marshal current_actions. ActiveflowCreate. err: %v", err)
	}

	tmpExecutedActions, err := json.Marshal(f.ExecutedActions)
	if err != nil {
		return fmt.Errorf("could not marshal executed_actions. ActiveflowCreate. err: %v", err)
	}

	tmpStackMap, err := json.Marshal(f.StackMap)
	if err != nil {
		return fmt.Errorf("could not marshal stack_map. ActiveflowCreate. err: %v", err)
	}

	// ts := h.util.TimeGetCurTime()
	_, err = stmt.ExecContext(ctx,
		f.ID.Bytes(),
		f.CustomerID.Bytes(),

		f.Status,
		f.FlowID.Bytes(),

		f.ReferenceType,
		f.ReferenceID.Bytes(),

		tmpStackMap,

		f.CurrentStackID.Bytes(),
		tmpCurrentAction,

		f.ForwardStackID.Bytes(),
		f.ForwardActionID.Bytes(),

		f.ExecuteCount,
		tmpExecutedActions,

		h.util.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. ActiveflowCreate. err: %v", err)
	}

	_ = h.activeflowUpdateToCache(ctx, f.ID)

	return nil
}

// activeflowGetFromDB gets the activeflow info from the db.
func (h *handler) activeflowGetFromDB(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", activeflowSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. activeflowGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. activeflowGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.activeflowGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// activeflowUpdateToCache gets the activeflow from the DB and update the cache.
func (h *handler) activeflowUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.activeflowGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.activeflowSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// activeFlowSetToCache sets the given activeflow to the cache
func (h *handler) activeflowSetToCache(ctx context.Context, flow *activeflow.Activeflow) error {
	if err := h.cache.ActiveflowSet(ctx, flow); err != nil {
		return err
	}

	return nil
}

// activeflowGetFromCache returns activeflow from the cache.
func (h *handler) activeflowGetFromCache(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {

	// get from cache
	res, err := h.cache.ActiveflowGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ActiveflowGet returns activeflow.
func (h *handler) ActiveflowGet(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {

	res, err := h.activeflowGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.activeflowGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.activeflowSetToCache(ctx, res)

	return res, nil
}

// ActiveflowGetWithLock returns activeflow.
func (h *handler) ActiveflowGetWithLock(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {

	// get data from the cache
	_, err := h.activeflowGetFromCache(ctx, id)
	if err == nil {
		// if not exist in the cache, update it to the cahce
		if errUpdate := h.activeflowUpdateToCache(ctx, id); errUpdate != nil {
			return nil, errUpdate
		}
	}

	// get with lock
	res, err := h.cache.ActiveflowGetWithLock(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ActiveflowReleaseLock releases the lock
func (h *handler) ActiveflowReleaseLock(ctx context.Context, id uuid.UUID) error {
	return h.cache.ActiveflowReleaseLock(ctx, id)
}

// ActiveflowGets returns flows.
func (h *handler) ActiveflowGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*activeflow.Activeflow, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, activeflowSelect)

	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "flow_id", "reference_id", "current_stack_id", "forward_stack_id", "forward_action_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
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
		return nil, fmt.Errorf("could not query. ActiveflowGets. err: %v", err)
	}
	defer rows.Close()

	res := []*activeflow.Activeflow{}
	for rows.Next() {
		u, err := h.activeflowGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. ActiveflowGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ActiveflowUpdate updates action related information.
func (h *handler) ActiveflowUpdate(ctx context.Context, af *activeflow.Activeflow) error {

	q := `
	update activeflows set
		current_stack_id = ?,
		current_action = ?,

		forward_stack_id = ?,
		forward_action_id = ?,

		stack_map = ?,

		execute_count = ?,
		executed_actions = ?,

		tm_update = ?
	where
		id = ?
	`

	tmpCurrentAction, err := json.Marshal(af.CurrentAction)
	if err != nil {
		return fmt.Errorf("could not marshal current_action. ActiveflowUpdateActionInfo. err: %v", err)
	}
	tmpStackMap, err := json.Marshal(af.StackMap)
	if err != nil {
		return fmt.Errorf("could not marshal stack_map. ActiveflowUpdateActionInfo. err: %v", err)
	}
	tmpExecutedActions, err := json.Marshal(af.ExecutedActions)
	if err != nil {
		return fmt.Errorf("could not marshal executed_actions. ActiveflowUpdateActionInfo. err: %v", err)
	}

	if _, err := h.db.Exec(q, af.CurrentStackID.Bytes(), tmpCurrentAction, af.ForwardStackID.Bytes(), af.ForwardActionID.Bytes(), tmpStackMap, af.ExecuteCount, tmpExecutedActions, h.util.TimeGetCurTime(), af.ID.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ActiveflowUpdateActionInfo. err: %v", err)
	}

	// set to the cache
	_ = h.activeflowUpdateToCache(ctx, af.ID)

	return nil
}

// ActiveflowDelete deletes the activeflow.
func (h *handler) ActiveflowDelete(ctx context.Context, id uuid.UUID) error {

	q := `
	update activeflows set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.util.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ActiveflowDelete. err: %v", err)
	}

	// set to the cache
	_ = h.activeflowUpdateToCache(ctx, id)

	return nil
}

// ActiveflowSetStatus sets the status.
func (h *handler) ActiveflowSetStatus(ctx context.Context, id uuid.UUID, status activeflow.Status) error {

	q := `
	update activeflows set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.util.TimeGetCurTime()
	if _, err := h.db.Exec(q, status, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ActiveflowSetStatus. err: %v", err)
	}

	// set to the cache
	_ = h.activeflowUpdateToCache(ctx, id)

	return nil
}
