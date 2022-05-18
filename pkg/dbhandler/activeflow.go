package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

const (
	// select query for flow get
	activeflowSelect = `
	select
		id,

		customer_id,
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
		?,
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

	_, err = stmt.ExecContext(ctx,
		f.ID.Bytes(),

		f.CustomerID.Bytes(),
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

		f.TMCreate,
		f.TMUpdate,
		f.TMDelete,
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

// activeflowUpdateToCache gets the flow from the DB and update the cache.
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

// activeFlowSetToCache sets the given callflow to the cache
func (h *handler) activeflowSetToCache(ctx context.Context, flow *activeflow.Activeflow) error {
	if err := h.cache.ActiveflowSet(ctx, flow); err != nil {
		return err
	}

	return nil
}

// activeflowGetFromCache returns flow from the cache if possible.
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

// ActiveflowGetsByCustomerID returns list of activeflows.
func (h *handler) ActiveflowGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*activeflow.Activeflow, error) {

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
	`, activeflowSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. ActiveflowGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	var res []*activeflow.Activeflow
	for rows.Next() {
		u, err := h.activeflowGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. ActiveflowGetsByCustomerID. err: %v", err)
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

	if _, err := h.db.Exec(q, af.CurrentStackID.Bytes(), tmpCurrentAction, af.ForwardStackID.Bytes(), af.ForwardActionID.Bytes(), tmpStackMap, af.ExecuteCount, tmpExecutedActions, GetCurTime(), af.ID.Bytes()); err != nil {
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
	and
		tm_delete != ?
	`

	ts := GetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes(), DefaultTimeStamp); err != nil {
		return fmt.Errorf("could not execute the query. ActiveflowDelete. err: %v", err)
	}

	// set to the cache
	_ = h.activeflowUpdateToCache(ctx, id)

	return nil
}
