package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/activeflow"
)

var (
	activeflowsTable  = "flow_activeflows"
	activeflowsFields = []string{
		string(activeflow.FieldID),
		string(activeflow.FieldCustomerID),

		string(activeflow.FieldStatus),
		string(activeflow.FieldFlowID),

		string(activeflow.FieldReferenceType),
		string(activeflow.FieldReferenceID),
		string(activeflow.FieldReferenceActiveflowID),

		string(activeflow.FieldStackMap),
		string(activeflow.FieldCurrentStackID),
		string(activeflow.FieldCurrentAction),

		string(activeflow.FieldForwardStackID),
		string(activeflow.FieldForwardActionID),

		string(activeflow.FieldExecuteCount),
		string(activeflow.FieldExecutedActions),

		string(activeflow.FieldTMCreate),
		string(activeflow.FieldTMUpdate),
		string(activeflow.FieldTMDelete),
	}
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
		&res.ReferenceActiveflowID,

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

func (h *handler) ActiveflowCreate(ctx context.Context, f *activeflow.Activeflow) error {
	now := h.util.TimeGetCurTime()

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

	sb := squirrel.
		Insert(activeflowsTable).
		Columns(activeflowsFields...).
		Values(
			f.ID.Bytes(),
			f.CustomerID.Bytes(),

			f.Status,
			f.FlowID.Bytes(),

			f.ReferenceType,
			f.ReferenceID.Bytes(),
			f.ReferenceActiveflowID.Bytes(),

			tmpStackMap,

			f.CurrentStackID.Bytes(),
			tmpCurrentAction,

			f.ForwardStackID.Bytes(),
			f.ForwardActionID.Bytes(),

			f.ExecuteCount,
			tmpExecutedActions,

			now,                                    // tm_create
			commondatabasehandler.DefaultTimeStamp, // tm_update
			commondatabasehandler.DefaultTimeStamp, // tm_delete
		).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ActiveflowCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ActiveflowCreate. err: %v", err)
	}

	_ = h.activeflowUpdateToCache(ctx, f.ID)
	return nil
}

// activeflowGetFromDB gets the activeflow info from the db.
func (h *handler) activeflowGetFromDB(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	query, args, err := squirrel.
		Select(activeflowsFields...).
		From(activeflowsTable).
		Where(squirrel.Eq{string(activeflow.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. activeflowGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. activeflowGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. activeflowGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.activeflowGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. activeflowGetFromDB. id: %s", id)
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

func (h *handler) ActiveflowGets(ctx context.Context, token string, size uint64, filters map[activeflow.Field]any) ([]*activeflow.Activeflow, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	sb := squirrel.
		Select(activeflowsFields...).
		From(activeflowsTable).
		Where(squirrel.Lt{string(activeflow.FieldTMCreate): token}).
		OrderBy(string(activeflow.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ActiveflowGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ActiveflowGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ActiveflowGets. err: %v", err)
	}
	defer rows.Close()

	res := []*activeflow.Activeflow{}
	for rows.Next() {
		u, err := h.activeflowGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ActiveflowGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ActiveflowGets. err: %v", err)
	}

	return res, nil
}

func (h *handler) ActiveflowUpdate(ctx context.Context, id uuid.UUID, fields map[activeflow.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[activeflow.FieldTMUpdate] = h.util.TimeGetCurTime()

	tmpFields := commondatabasehandler.PrepareUpdateFields(fields)
	q := squirrel.Update(activeflowsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{"id": id.Bytes()})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ActiveflowUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("ActiveflowUpdate: exec failed: %w", err)
	}

	_ = h.activeflowUpdateToCache(ctx, id)
	return nil
}

func (h *handler) ActiveflowDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.util.TimeGetCurTime()

	fields := map[activeflow.Field]any{
		activeflow.FieldTMUpdate: ts,
		activeflow.FieldTMDelete: ts,
	}

	tmpFields := commondatabasehandler.PrepareUpdateFields(fields)
	sb := squirrel.Update(activeflowsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(activeflow.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ActiveflowDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("ActiveflowDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.activeflowUpdateToCache(ctx, id)
	return nil
}
