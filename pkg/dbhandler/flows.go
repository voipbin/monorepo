package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/flow"
)

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
func (h *handler) FlowSetToCache(ctx context.Context, flow *flow.Flow) error {
	if err := h.cache.FlowSet(ctx, flow); err != nil {
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

func (h *handler) FlowCreate(ctx context.Context, flow *flow.Flow) error {

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

	tmpActions, err := json.Marshal(flow.Actions)
	if err != nil {
		return fmt.Errorf("could not marshal actions. FlowCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		flow.ID.Bytes(),
		flow.UserID,

		flow.Name,
		flow.Detail,

		tmpActions,

		flow.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. FlowCreate. err: %v", err)
	}

	h.FlowUpdateToCache(ctx, flow.ID)

	return nil
}

// FlowGet returns flow.
func (h *handler) FlowGetFromDB(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {

	// prepare
	q := `
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
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. FlowGet. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. FlowGet. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

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
		return nil, fmt.Errorf("could not scan the row. FlowGet. err: %v", err)
	}

	if err := json.Unmarshal([]byte(actions), &res.Actions); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. FlowGet. err: %v", err)
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

	// set to the cache
	h.FlowSetToCache(ctx, res)

	return res, nil
}
