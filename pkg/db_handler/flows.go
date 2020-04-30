package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	flow "gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow"
)

func (h *handler) FlowCreate(ctx context.Context, flow *flow.Flow) error {

	q := `insert into fm_flows(
		id,
		revision,
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
		flow.Revision.Bytes(),

		flow.Name,
		flow.Detail,

		tmpActions,

		flow.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. FlowCreate. err: %v", err)
	}

	return nil
}

// FlowGet returns flow.
func (h *handler) FlowGet(ctx context.Context, id, revision uuid.UUID) (*flow.Flow, error) {

	// prepare
	q := `
	select
		id,
		revision,

		name,
		detail,

		actions,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete
	from
		fm_flows
	where
		id = ?
		and revision = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. FlowGet. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes(), revision.Bytes())
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
		&res.Revision,

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
