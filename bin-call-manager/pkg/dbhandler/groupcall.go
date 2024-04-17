package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-call-manager/models/groupcall"
)

const (
	// select query for call get
	groupcallSelect = `
	select
		id,
		customer_id,

		status,
		flow_id,

		source,
		destinations,

		master_call_id,
		master_groupcall_id,

		ring_method,
		answer_method,

		answer_call_id,
		call_ids,

		answer_groupcall_id,
		groupcall_ids,

		call_count,
		groupcall_count,
		dial_index,

		tm_create,
		tm_update,
		tm_delete
	from
		groupcalls
	`
)

// groupcallGetFromRow gets the groupcall from the row.
func (h *handler) groupcallGetFromRow(row *sql.Rows) (*groupcall.Groupcall, error) {
	var source sql.NullString
	var destinations sql.NullString
	var callIDs sql.NullString
	var groupcallIDs sql.NullString

	res := &groupcall.Groupcall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Status,
		&res.FlowID,

		&source,
		&destinations,

		&res.MasterCallID,
		&res.MasterGroupcallID,

		&res.RingMethod,
		&res.AnswerMethod,

		&res.AnswerCallID,
		&callIDs,

		&res.AnswerGroupcallID,
		&groupcallIDs,

		&res.CallCount,
		&res.GroupcallCount,
		&res.DialIndex,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. groupcallGetFromRow. err: %v", err)
	}

	// Source
	if source.Valid {
		if err := json.Unmarshal([]byte(source.String), &res.Source); err != nil {
			return nil, fmt.Errorf("could not unmarshal the source. groupcallGetFromRow. err: %v", err)
		}
	} else {
		res.Source = &commonaddress.Address{}
	}

	// destinations
	if destinations.Valid {
		if err := json.Unmarshal([]byte(destinations.String), &res.Destinations); err != nil {
			return nil, fmt.Errorf("could not unmarshal the destinations. groupcallGetFromRow. err: %v", err)
		}
	} else {
		res.Destinations = []commonaddress.Address{}
	}

	// CallIDs
	if callIDs.Valid {
		if err := json.Unmarshal([]byte(callIDs.String), &res.CallIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the call_ids. groupcallGetFromRow. err: %v", err)
		}
	} else {
		res.CallIDs = []uuid.UUID{}
	}

	// GroupcallIDs
	if groupcallIDs.Valid {
		if err := json.Unmarshal([]byte(groupcallIDs.String), &res.GroupcallIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the groupcall_ids. groupcallGetFromRow. err: %v", err)
		}
	} else {
		res.GroupcallIDs = []uuid.UUID{}
	}

	return res, nil
}

// GroupcallCreate sets groupcall.
func (h *handler) GroupcallCreate(ctx context.Context, data *groupcall.Groupcall) error {

	q := `insert into groupcalls(
		id,
		customer_id,

		status,
		flow_id,

		source,
		destinations,

		master_call_id,
		master_groupcall_id,

		ring_method,
		answer_method,

		answer_call_id,
		call_ids,

		answer_groupcall_id,
		groupcall_ids,

		call_count,
		groupcall_count,
		dial_index,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?,
		?, ?,
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?,
		?, ?, ?
		)`

	if data.Source == nil {
		data.Source = &commonaddress.Address{}
	}
	tmpSource, err := json.Marshal(data.Source)
	if err != nil {
		return errors.Wrap(err, "could not marshal the source. GroupcallCreate.")
	}

	if data.Destinations == nil {
		data.Destinations = []commonaddress.Address{}
	}
	tmpDestinations, err := json.Marshal(data.Destinations)
	if err != nil {
		return errors.Wrap(err, "could not marshal the destinations. GroupcallCreate.")
	}

	if data.CallIDs == nil {
		data.CallIDs = []uuid.UUID{}
	}
	tmpCallIDs, err := json.Marshal(data.CallIDs)
	if err != nil {
		return errors.Wrap(err, "could not marshal the call_ids. GroupcallCreate.")
	}

	if data.GroupcallIDs == nil {
		data.GroupcallIDs = []uuid.UUID{}
	}
	tmpGroupcallIDs, err := json.Marshal(data.GroupcallIDs)
	if err != nil {
		return errors.Wrap(err, "could not marshal the groupcall_ids. GroupcallCreate.")
	}

	_, err = h.db.Exec(q,
		data.ID.Bytes(),
		data.CustomerID.Bytes(),

		data.Status,
		data.FlowID.Bytes(),

		tmpSource,
		tmpDestinations,

		data.MasterCallID.Bytes(),
		data.MasterGroupcallID.Bytes(),

		data.RingMethod,
		data.AnswerMethod,

		data.AnswerCallID.Bytes(),
		tmpCallIDs,

		data.AnswerGroupcallID.Bytes(),
		tmpGroupcallIDs,

		data.CallCount,
		data.GroupcallCount,
		data.DialIndex,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return errors.Wrap(err, "could not execute. GroupcallCreate.")
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, data.ID)

	return nil
}

// GroupcallGet returns groupcall.
func (h *handler) GroupcallGet(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {

	res, err := h.groupcallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.groupcallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.groupcallSetToCache(ctx, res)

	return res, nil
}

// GroupcallGets returns a list of groupcalls.
func (h *handler) GroupcallGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*groupcall.Groupcall, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, groupcallSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "flow_id", "master_call_id", "master_groupcall_id", "answer_call_id", "answer_groupcall_id":
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
		return nil, fmt.Errorf("could not query. GroupcallGets. err: %v", err)
	}
	defer rows.Close()

	res := []*groupcall.Groupcall{}
	for rows.Next() {
		u, err := h.groupcallGetFromRow(rows)
		if err != nil {
			return nil, errors.Wrap(err, "Could not get data. GroupcallGets.")
		}

		res = append(res, u)
	}

	return res, nil
}

// GroupcallSetAnswerCallID updates the answer call id.
func (h *handler) GroupcallSetAnswerCallID(ctx context.Context, id uuid.UUID, answerCallID uuid.UUID) error {
	// prepare
	q := `
	update
		groupcalls
	set
		answer_call_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, answerCallID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallSetAnswerCallID. err: %v", err)
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, id)

	return nil
}

// GroupcallSetAnswerGroupcallID updates the answer groupcall id.
func (h *handler) GroupcallSetAnswerGroupcallID(ctx context.Context, id uuid.UUID, answerGroupcallID uuid.UUID) error {
	// prepare
	q := `
	update
		groupcalls
	set
		answer_groupcall_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, answerGroupcallID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallSetAnswerGroupcallID. err: %v", err)
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, id)

	return nil
}

// callGetFromCache returns call from the cache.
func (h *handler) groupcallGetFromCache(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {

	// get from cache
	res, err := h.cache.GroupcallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// groupcallGetFromDB returns groupcall from the DB.
func (h *handler) groupcallGetFromDB(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", groupcallSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. groupcallGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.groupcallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. groupcallGetFromDB, err: %v", err)
	}

	return res, nil
}

// groupcallUpdateToCache gets the groupcall from the DB and update the cache.
func (h *handler) groupcallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.groupcallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.groupcallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// groupcallSetToCache sets the given groupcall to the cache
func (h *handler) groupcallSetToCache(ctx context.Context, data *groupcall.Groupcall) error {
	if err := h.cache.GroupcallSet(ctx, data); err != nil {
		return err
	}

	return nil
}

// GroupcallDelete deletes the groupcall
func (h *handler) GroupcallDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update groupcalls set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallDelete. err: %v", err)
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, id)

	return nil
}

// GroupcallDecreaseCallCount decreases the call count
func (h *handler) GroupcallDecreaseCallCount(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update groupcalls set
		call_count = call_count - 1,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallDecreaseCallCount. err: %v", err)
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, id)

	return nil
}

// GroupcallDecreaseGroupcallCount decreases the groupcall count
func (h *handler) GroupcallDecreaseGroupcallCount(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update groupcalls set
		groupcall_count = groupcall_count - 1,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallDecreaseGroupcallCount. err: %v", err)
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, id)

	return nil
}

// GroupcallSetStatus updates the status
func (h *handler) GroupcallSetStatus(ctx context.Context, id uuid.UUID, status groupcall.Status) error {
	//prepare
	q := `
	update groupcalls set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, status, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallSetStatus. err: %v", err)
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, id)

	return nil
}

// GroupcallSetCallIDsAndCallCountAndDialIndex updates the call_ids and call_count and dial_index
func (h *handler) GroupcallSetCallIDsAndCallCountAndDialIndex(ctx context.Context, id uuid.UUID, callIDs []uuid.UUID, callCount int, dialIndex int) error {
	//prepare
	q := `
	update groupcalls set
		call_ids = ?,
		call_count = ?,
		dial_index = ?,
		tm_update = ?
	where
		id = ?
	`

	if callIDs == nil {
		callIDs = []uuid.UUID{}
	}
	tmpCallIDs, err := json.Marshal(callIDs)
	if err != nil {
		return errors.Wrap(err, "could not marshal the call_ids. GroupcallSetCallIDsAndCallCountAndDialIndex.")
	}

	ts := h.utilHandler.TimeGetCurTime()
	_, err = h.db.Exec(q, tmpCallIDs, callCount, dialIndex, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallSetCallIDsAndCallCountAndDialIndex. err: %v", err)
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, id)

	return nil
}

// GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex updates the call_ids and call_count and dial_index
func (h *handler) GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx context.Context, id uuid.UUID, groupcallIDs []uuid.UUID, groupcallCount int, dialIndex int) error {
	//prepare
	q := `
	update groupcalls set
		groupcall_ids = ?,
		groupcall_count = ?,
		dial_index = ?,
		tm_update = ?
	where
		id = ?
	`

	if groupcallIDs == nil {
		groupcallIDs = []uuid.UUID{}
	}
	tmpGroupcallIDs, err := json.Marshal(groupcallIDs)
	if err != nil {
		return errors.Wrap(err, "could not marshal the call_ids. GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex.")
	}

	ts := h.utilHandler.TimeGetCurTime()
	_, err = h.db.Exec(q, tmpGroupcallIDs, groupcallCount, dialIndex, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex. err: %v", err)
	}

	// update the cache
	_ = h.groupcallUpdateToCache(ctx, id)

	return nil
}
