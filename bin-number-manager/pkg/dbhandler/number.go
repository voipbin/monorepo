package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
)

const (
	numberSelect = `
	select
		id,
		number,
		customer_id,

		call_flow_id,
		message_flow_id,

		name,
		detail,

		provider_name,
		provider_reference_id,

		status,

		t38_enabled,
		emergency_enabled,

		tm_purchase,
		tm_renew,

		tm_create,
		tm_update,
		tm_delete

	from
		number_numbers
	`
)

// numberGetFromRow gets the number from the row.
func (h *handler) numberGetFromRow(row *sql.Rows) (*number.Number, error) {
	res := &number.Number{}
	if err := row.Scan(
		&res.ID,
		&res.Number,
		&res.CustomerID,

		&res.CallFlowID,
		&res.MessageFlowID,

		&res.Name,
		&res.Detail,

		&res.ProviderName,
		&res.ProviderReferenceID,

		&res.Status,

		&res.T38Enabled,
		&res.EmergencyEnabled,

		&res.TMPurchase,
		&res.TMRenew,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. numberGetFromRow. err: %v", err)
	}

	return res, nil
}

// NumberCreate creates a new number record.
func (h *handler) NumberCreate(ctx context.Context, n *number.Number) error {
	q := `insert into number_numbers(
		id,
		number,
		customer_id,

		call_flow_id,
		message_flow_id,

		name,
		detail,

		provider_name,
		provider_reference_id,

		status,

		t38_enabled,
		emergency_enabled,

		tm_purchase,
		tm_renew,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?, ?,
		?, ?,
		?,
		?, ?,
		?, ?,
		?, ?, ?
		)`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q,
		n.ID.Bytes(),
		n.Number,
		n.CustomerID.Bytes(),

		n.CallFlowID.Bytes(),
		n.MessageFlowID.Bytes(),

		n.Name,
		n.Detail,

		n.ProviderName,
		n.ProviderReferenceID,

		n.Status,

		n.T38Enabled,
		n.EmergencyEnabled,

		ts,
		ts,

		ts,
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. NumberCreate. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, n.ID)

	return nil
}

// NumberGetFromCacheByNumber returns number from the cache by number.
func (h *handler) NumberGetFromCacheByNumber(ctx context.Context, numb string) (*number.Number, error) {

	// get from cache
	res, err := h.cache.NumberGetByNumber(ctx, numb)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// numberGetFromCache returns number from the cache.
func (h *handler) numberGetFromCache(ctx context.Context, id uuid.UUID) (*number.Number, error) {

	// get from cache
	res, err := h.cache.NumberGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// numberSetToCache sets the given number to the cache
func (h *handler) numberSetToCache(ctx context.Context, num *number.Number) error {
	if err := h.cache.NumberSet(ctx, num); err != nil {
		return err
	}

	return nil
}

// numberUpdateToCache gets the number from the DB and update the cache.
func (h *handler) numberUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.numberGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.numberSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// numberGetFromDB returns number info from the DB.
func (h *handler) numberGetFromDB(ctx context.Context, id uuid.UUID) (*number.Number, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", numberSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. numberGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.numberGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get number. numberGetFromDB, err: %v", err)
	}

	return res, nil
}

// NumberGet returns number.
func (h *handler) NumberGet(ctx context.Context, id uuid.UUID) (*number.Number, error) {

	res, err := h.numberGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.numberGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.numberSetToCache(ctx, res)

	return res, nil
}

func (h *handler) numberGetsMergeFilters(query string, values []interface{}, filters map[string]string) (string, []interface{}) {

	for k, v := range filters {
		switch k {
		case "customer_id", "call_flow_id", "message_flow_id":
			query = fmt.Sprintf("%s and %s = ?", query, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				query = fmt.Sprintf("%s and tm_delete >= ?", query)
				values = append(values, DefaultTimeStamp)
			}

		default:
			query = fmt.Sprintf("%s and %s = ?", query, k)
			values = append(values, v)
		}
	}

	return query, values
}

// NumberGets returns a list of numbers.
func (h *handler) NumberGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*number.Number, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, numberSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	q, values = h.numberGetsMergeFilters(q, values, filters)
	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))
	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGets err: %v", err)
	}
	defer rows.Close()

	res := []*number.Number{}
	for rows.Next() {
		u, err := h.numberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. NumberGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil

}

// NumberDelete sets the delte timestamp.
func (h *handler) NumberDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
		update
			number_numbers
		set
			status = ?,
			tm_update = ?,
			tm_delete = ?
		where
			id = ?
		`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, string(number.StatusDeleted), ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. NumberDelete. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}

// NumberUpdateInfo updates basic number information.
func (h *handler) NumberUpdateInfo(ctx context.Context, id uuid.UUID, callflowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) error {
	q := `
	update number_numbers set
		call_flow_id = ?,
		message_flow_id = ?,
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		callflowID.Bytes(),
		messageFlowID.Bytes(),
		name,
		detail,
		h.utilHandler.TimeGetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. NumberUpdateBasicInfo. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}

// NumberUpdateFlowID updates number's flow id.
func (h *handler) NumberUpdateFlowID(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID) error {
	q := `
	update number_numbers set
		call_flow_id = ?,
		message_flow_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		callFlowID.Bytes(),
		messageFlowID.Bytes(),
		h.utilHandler.TimeGetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. NumberUpdateFlowID. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}

// NumberUpdateCallFlowID updates call_flow_id.
func (h *handler) NumberUpdateCallFlowID(ctx context.Context, id uuid.UUID, flowID uuid.UUID) error {
	q := `
	update number_numbers set
		call_flow_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		flowID.Bytes(),
		h.utilHandler.TimeGetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. NumberUpdateFlowID. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}

// NumberUpdateMessageFlowID updates message_flow_id.
func (h *handler) NumberUpdateMessageFlowID(ctx context.Context, id uuid.UUID, flowID uuid.UUID) error {
	q := `
	update number_numbers set
		message_flow_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		flowID.Bytes(),
		h.utilHandler.TimeGetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. NumberUpdateMessageFlowID. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}

// NumberUpdateTMRenew updates the tm_renew.
func (h *handler) NumberUpdateTMRenew(ctx context.Context, id uuid.UUID) error {
	q := `
	update number_numbers set
		tm_renew = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q,
		ts,
		ts,
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. NumberUpdateTMRenew. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}

// NumberGetsByTMRenew returns a list of numbers.
func (h *handler) NumberGetsByTMRenew(ctx context.Context, tmRenew string, size uint64, filters map[string]string) ([]*number.Number, error) {
	// prepare
	q := fmt.Sprintf(`%s
		where
			tm_renew < ?
		`, numberSelect)

	values := []interface{}{
		tmRenew,
	}

	// merge filters
	q, values = h.numberGetsMergeFilters(q, values, filters)

	// complete the query
	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGetsByTMRenew. err: %v", err)
	}
	defer rows.Close()

	res := []*number.Number{}
	for rows.Next() {
		u, err := h.numberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. NumberGetsByTMRenew, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil

}
