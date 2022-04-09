package dbhandler

import (
	context "context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
)

const (
	// select query for outplan get
	outplanSelect = `
	select
		id,
		customer_id,

		name,
		detail,

		actions,
		source,
		dial_timeout,
		end_handle,
		try_interval,

		max_try_count_0,
		max_try_count_1,
		max_try_count_2,
		max_try_count_3,
		max_try_count_4,

		tm_create,
		tm_update,
		tm_delete
	from
		outplans
	`
)

// outplanGetFromRow gets the outplan from the row.
func (h *handler) outplanGetFromRow(row *sql.Rows) (*outplan.Outplan, error) {
	var actions string
	var source string

	res := &outplan.Outplan{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,

		&actions,
		&source,
		&res.DialTimeout,
		&res.EndHandle,
		&res.TryInterval,

		&res.MaxTryCount0,
		&res.MaxTryCount1,
		&res.MaxTryCount2,
		&res.MaxTryCount3,
		&res.MaxTryCount4,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. outplanGetFromRow. err: %v", err)
	}

	if errActions := json.Unmarshal([]byte(actions), &res.Actions); errActions != nil {
		return nil, fmt.Errorf("could not unmarshal the actions. outplanGetFromRow. err: %v", errActions)
	}

	if errSource := json.Unmarshal([]byte(source), &res.Source); errSource != nil {
		return nil, fmt.Errorf("could not unmarshal the source. outplanGetFromRow. err: %v", errSource)
	}

	return res, nil
}

// OutplanCreate insert a new plan record
func (h *handler) OutplanCreate(ctx context.Context, t *outplan.Outplan) error {

	q := `insert into outplans(
		id,
		customer_id,

		name,
		detail,

		actions,
		source,
		dial_timeout,
		end_handle,
		try_interval,

		max_try_count_0,
		max_try_count_1,
		max_try_count_2,
		max_try_count_3,
		max_try_count_4,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. OutplanCreate. err: %v", err)
	}
	defer stmt.Close()

	actions, err := json.Marshal(t.Actions)
	if err != nil {
		return fmt.Errorf("could not marshal actions. OutplanCreate. err: %v", err)
	}

	source, err := json.Marshal(t.Source)
	if err != nil {
		return fmt.Errorf("could not marshal source. OutplanCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		t.ID.Bytes(),
		t.CustomerID.Bytes(),

		t.Name,
		t.Detail,

		actions,
		source,
		t.DialTimeout,
		t.EndHandle,
		t.TryInterval,

		t.MaxTryCount0,
		t.MaxTryCount1,
		t.MaxTryCount2,
		t.MaxTryCount3,
		t.MaxTryCount4,

		t.TMCreate,
		t.TMUpdate,
		t.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. OutplanCreate. err: %v", err)
	}

	_ = h.outplanUpdateToCache(ctx, t.ID)

	return nil
}

// outplanUpdateToCache gets the outplan from the DB and update the cache.
func (h *handler) outplanUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.outplanGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.outplanSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// outplanSetToCache sets the given outplan to the cache
func (h *handler) outplanSetToCache(ctx context.Context, f *outplan.Outplan) error {
	if err := h.cache.OutplanSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// outplanGetFromCache returns outplan from the cache if possible.
func (h *handler) outplanGetFromCache(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {

	// get from cache
	res, err := h.cache.OutplanGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outplanGetFromDB gets the outplan info from the db.
func (h *handler) outplanGetFromDB(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", outplanSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. outplanGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. outplanGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.outplanGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// OutplanDelete deletes the given outplan
func (h *handler) OutplanDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update outplans set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := GetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutplanDelete. err: %v", err)
	}

	// update cache
	_ = h.outplanUpdateToCache(ctx, id)

	return nil
}

// OutplanGet returns outplan.
func (h *handler) OutplanGet(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {

	res, err := h.outplanGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.outplanGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.outplanSetToCache(ctx, res)

	return res, nil
}

// OutplanGetsByCustomerID returns list of outplans.
func (h *handler) OutplanGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outplan.Outplan, error) {

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
	`, outplanSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutplanGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	var res []*outplan.Outplan
	for rows.Next() {
		u, err := h.outplanGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. OutplanGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// OutplanUpdateBasicInfo updates outplan's basic information.
func (h *handler) OutplanUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	q := `
	update outplans set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, name, detail, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutplanUpdateBasicInfo. err: %v", err)
	}

	// set to the cache
	_ = h.outplanUpdateToCache(ctx, id)

	return nil
}

// OutplanUpdateActionInfo updates outplan's action related information.
func (h *handler) OutplanUpdateActionInfo(ctx context.Context, id uuid.UUID, actions []fmaction.Action, source *cmaddress.Address, endHandle outplan.EndHandle) error {
	q := `
	update outplans set
		actions = ?,
		source = ?,
		end_handle = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpActions, err := json.Marshal(actions)
	if err != nil {
		return fmt.Errorf("could not marshal the actions. OutplanUpdateActionInfo. err: %v", err)
	}

	tmpSource, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("could not marshal the source. OutplanUpdateActionInfo. err: %v", err)
	}

	if _, err := h.db.Exec(q, tmpActions, tmpSource, endHandle, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutplanUpdateActionInfo. err: %v", err)
	}

	// set to the cache
	_ = h.outplanUpdateToCache(ctx, id)

	return nil
}

// OutplanUpdateDialInfo updates outplan's action related information.
func (h *handler) OutplanUpdateDialInfo(ctx context.Context, id uuid.UUID, dialTimeout, tryInterval, maxTryCount0, maxTryCount1, maxTryCount2, maxTryCount3, maxTryCount4 int) error {
	q := `
	update outplans set
		dial_timeout = ?,
		try_interval = ?,
		max_try_count_0 = ?,
		max_try_count_1 = ?,
		max_try_count_2 = ?,
		max_try_count_3 = ?,
		max_try_count_4 = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, dialTimeout, tryInterval, maxTryCount0, maxTryCount1, maxTryCount2, maxTryCount3, maxTryCount4, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutplanUpdateActionInfo. err: %v", err)
	}

	// set to the cache
	_ = h.outplanUpdateToCache(ctx, id)

	return nil
}
