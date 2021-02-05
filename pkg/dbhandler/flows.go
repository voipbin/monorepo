package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
)

const (
	flowSelect = `
		select
		id,
		user_id,

		name,
		detail,

		actions,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		flows
	`
)

// flowGetFromRow gets the flow from the row.
func (h *handler) flowGetFromRow(row *sql.Rows) (*flow.Flow, error) {
	var actions string

	res := &flow.Flow{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,

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

	return res, nil
}

// FlowUpdateToCache gets the flow from the DB and update the cache.
func (h *handler) FlowUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.FlowGetFromDB(ctx, id)
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

// FlowGetFromCache returns flow from the cache if possible.
func (h *handler) FlowGetFromCache(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {

	// get from cache
	res, err := h.cache.FlowGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// FlowDeleteCache deletes cache
func (h *handler) FlowDeleteCache(ctx context.Context, id uuid.UUID) error {

	// delete from cache
	err := h.cache.FlowDel(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) FlowCreate(ctx context.Context, f *flow.Flow) error {

	q := `insert into flows(
		id,
		user_id,
		name,
		detail,

		actions,

		tm_create
	) values(
		?, ?, ?, ?,
		?,
		?
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
		f.UserID,

		f.Name,
		f.Detail,

		tmpActions,

		f.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. FlowCreate. err: %v", err)
	}

	h.FlowUpdateToCache(ctx, f.ID)

	return nil
}

// FlowGetFromDB gets the flow info from the db.
func (h *handler) FlowGetFromDB(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {

	// prepare
	q := fmt.Sprintf("%s where tm_delete is null and id = ?", flowSelect)

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

	if row.Next() == false {
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

	res, err := h.FlowGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.FlowGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	if res.TMDelete == "" {
		// set to the cache
		h.FlowSetToCache(ctx, res)
	}

	return res, nil
}

// FlowGetsByUserID returns list of flows.
func (h *handler) FlowGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*flow.Flow, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete is null
			and user_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, flowSelect)

	rows, err := h.db.Query(q, userID, token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. FlowGetsByUserID. err: %v", err)
	}
	defer rows.Close()

	var res []*flow.Flow
	for rows.Next() {
		u, err := h.flowGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. FlowGetsByUserID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// FlowUpdate updates the most of flow information.
// except permenant info(i.e. id, timestamp, etc)
func (h *handler) FlowUpdate(ctx context.Context, f *flow.Flow) error {
	q := fmt.Sprintf(`
	update flows set
		name = ?,
		detail = ?,
		actions = ?,
		tm_update = ?
	where
		id = ?
	`)

	tmpActions, err := json.Marshal(f.Actions)
	if err != nil {
		return fmt.Errorf("could not marshal actions. FlowUpdate. err: %v", err)
	}

	if _, err := h.db.Exec(q, f.Name, f.Detail, tmpActions, getCurTime(), f.ID.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. FlowUpdate. err: %v", err)
	}

	// set to the cache
	h.FlowUpdateToCache(ctx, f.ID)

	return nil
}

// FlowDelete deletes the given flow
func (h *handler) FlowDelete(ctx context.Context, id uuid.UUID) error {
	q := fmt.Sprintf(`
	update flows set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`)

	if _, err := h.db.Exec(q, getCurTime(), getCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. FlowDelete. err: %v", err)
	}

	// delete cache
	h.FlowDeleteCache(ctx, id)

	return nil
}
