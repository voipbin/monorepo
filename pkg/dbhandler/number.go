package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
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

		tm_create,
		tm_update,
		tm_delete

	from
		numbers
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
	q := `insert into numbers(
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
		?,
		?, ?, ?
		)`

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

		n.TMPurchase,

		h.util.GetCurTime(),
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

// numberSetToCacheByNumber sets the given number to the cache
func (h *handler) numberSetToCacheByNumber(ctx context.Context, num *number.Number) error {
	if err := h.cache.NumberSetByNumber(ctx, num); err != nil {
		return err
	}

	return nil
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

	if err := h.numberSetToCacheByNumber(ctx, num); err != nil {
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

// numberGetFromDBByNumber returns number info from the DB by number.
func (h *handler) numberGetFromDBByNumber(ctx context.Context, numb string) (*number.Number, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			number = ?
			and tm_delete >= ?
	`, numberSelect)

	row, err := h.db.Query(q, numb, DefaultTimeStamp)
	if err != nil {
		return nil, fmt.Errorf("could not query. numberGetFromDBByNumber. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.numberGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get number. numberGetFromDBByNumber, err: %v", err)
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

// NumberGetByNumber returns number by number.
func (h *handler) NumberGetByNumber(ctx context.Context, numb string) (*number.Number, error) {

	res, err := h.NumberGetFromCacheByNumber(ctx, numb)
	if err == nil {
		return res, nil
	}

	res, err = h.numberGetFromDBByNumber(ctx, numb)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.numberSetToCacheByNumber(ctx, res)

	return res, nil
}

// NumberGets returns a list of numbers.
func (h *handler) NumberGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*number.Number, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			customer_id = ?
			and tm_create < ?
			and tm_delete >= ?
		order by
			tm_create
		desc limit ?
		`, numberSelect)

	rows, err := h.db.Query(q, customerID.Bytes(), token, DefaultTimeStamp, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGets. err: %v", err)
	}
	defer rows.Close()

	res := []*number.Number{}
	for rows.Next() {
		u, err := h.numberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. NumberGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// NumberGetsByCallFlowID returns a list of numbers by call_flow_id.
func (h *handler) NumberGetsByCallFlowID(ctx context.Context, flowID uuid.UUID, size uint64, token string) ([]*number.Number, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			call_flow_id = ?
			and tm_create < ?
			and tm_delete >= ?
		order by
			tm_create desc
		limit ?
		`,
		numberSelect)

	rows, err := h.db.Query(q, flowID.Bytes(), token, DefaultTimeStamp, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGetsByCallFlowID. err: %v", err)
	}
	defer rows.Close()

	res := []*number.Number{}
	for rows.Next() {
		u, err := h.numberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. NumberGetsByCallFlowID, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// NumberGetsByMessageFlowID returns a list of numbers by message_flow_id.
func (h *handler) NumberGetsByMessageFlowID(ctx context.Context, flowID uuid.UUID, size uint64, token string) ([]*number.Number, error) {

	// prepare
	q := fmt.Sprintf(`%s
		where
			message_flow_id = ?
			and tm_create < ?
			and tm_delete >= ?
		order by
			tm_create desc
		limit ?
		`,
		numberSelect)

	rows, err := h.db.Query(q, flowID.Bytes(), token, DefaultTimeStamp, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGetsByMessageFlowID. err: %v", err)
	}
	defer rows.Close()

	res := []*number.Number{}
	for rows.Next() {
		u, err := h.numberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. NumberGetsByMessageFlowID, err: %v", err)
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
			numbers
		set
			status = ?,
			tm_update = ?,
			tm_delete = ?
		where
			id = ?
		`

	ts := h.util.GetCurTime()
	_, err := h.db.Exec(q, string(number.StatusDeleted), ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. NumberDelete. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}

// NumberUpdateBasicInfo updates basic number information.
func (h *handler) NumberUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	q := `
	update numbers set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		name,
		detail,
		h.util.GetCurTime(),
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
func (h *handler) NumberUpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) error {
	q := `
	update numbers set
		call_flow_id = ?,
		message_flow_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		callFlowID.Bytes(),
		messageFlowID.Bytes(),
		h.util.GetCurTime(),
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
func (h *handler) NumberUpdateCallFlowID(ctx context.Context, id, flowID uuid.UUID) error {
	q := `
	update numbers set
		call_flow_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		flowID.Bytes(),
		h.util.GetCurTime(),
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
func (h *handler) NumberUpdateMessageFlowID(ctx context.Context, id, flowID uuid.UUID) error {
	q := `
	update numbers set
		message_flow_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q,
		flowID.Bytes(),
		h.util.GetCurTime(),
		id.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. NumberUpdateMessageFlowID. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}
