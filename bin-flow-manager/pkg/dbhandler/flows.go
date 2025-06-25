package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-flow-manager/models/flow"
)

var (
	flowsTable  = "flow_flows"
	flowsFields = []string{
		string(flow.FieldID),
		string(flow.FieldCustomerID),

		string(flow.FieldType),

		string(flow.FieldName),
		string(flow.FieldDetail),

		string(flow.FieldActions),

		string(flow.FieldTMCreate),
		string(flow.FieldTMUpdate),
		string(flow.FieldTMDelete),
	}
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
	now := h.util.TimeGetCurTime()

	tmpActions, err := json.Marshal(f.Actions)
	if err != nil {
		return fmt.Errorf("could not marshal current_actions. FlowCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(flowsTable).
		Columns(flowsFields...).
		Values(
			f.ID.Bytes(),
			f.CustomerID.Bytes(),

			f.Type,

			f.Name,
			f.Detail,

			tmpActions,

			now,                                    // tm_create
			commondatabasehandler.DefaultTimeStamp, // tm_update
			commondatabasehandler.DefaultTimeStamp, // tm_delete
		).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. FlowCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
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

func (h *handler) flowGetFromDB(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	query, args, err := squirrel.
		Select(flowsFields...).
		From(flowsTable).
		Where(squirrel.Eq{string(flow.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. flowGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. flowGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. flowGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.flowGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. flowGetFromDB. id: %s", id)
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

func (h *handler) FlowGets(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	sb := squirrel.
		Select(flowsFields...).
		From(flowsTable).
		Where(squirrel.Lt{string(flow.FieldTMCreate): token}).
		OrderBy(string(flow.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. FlowGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. FlowGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. FlowGets. err: %v", err)
	}
	defer rows.Close()

	res := []*flow.Flow{}
	for rows.Next() {
		u, err := h.flowGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. FlowGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. FlowGets. err: %v", err)
	}

	return res, nil
}

func (h *handler) FlowUpdate(ctx context.Context, id uuid.UUID, fields map[flow.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[flow.FieldTMUpdate] = h.util.TimeGetCurTime()

	return h.flowUpdate(ctx, id, fields)
}

func (h *handler) flowUpdate(ctx context.Context, id uuid.UUID, fields map[flow.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	tmpFields := commondatabasehandler.PrepareUpdateFields(fields)
	q := squirrel.Update(flowsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{"id": id.Bytes()})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("FlowUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("FlowUpdate: exec failed: %w", err)
	}

	_ = h.flowUpdateToCache(ctx, id)
	return nil
}

// FlowDelete deletes the given flow
func (h *handler) FlowDelete(ctx context.Context, id uuid.UUID) error {

	now := h.util.TimeGetCurTime()
	fields := map[flow.Field]any{
		flow.FieldTMDelete: now,
		flow.FieldTMUpdate: now,
	}

	if errUpdate := h.flowUpdate(ctx, id, fields); errUpdate != nil {
		return fmt.Errorf("could not update flow for delete. FlowDelete. err: %v", errUpdate)
	}

	return nil
}
