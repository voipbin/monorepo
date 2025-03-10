package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"monorepo/bin-queue-manager/models/queuecall"
)

const (
	// select query for  queuecall get
	queuecallSelect = `
	select
		id,
		customer_id,
		queue_id,

		reference_type,
		reference_id,
		reference_activeflow_id,

		forward_action_id,
 		confbridge_id,

		source,
		routing_method,
		tag_ids,

		status,
		service_agent_id,

		timeout_wait,
		timeout_service,

		duration_waiting,
		duration_service,

		tm_create,
		tm_service,
		tm_update,
		tm_end,
		tm_delete
	from
		queue_queuecalls
	`
)

// queuecallGetFromRow gets the  queuecall from the row.
func (h *handler) queuecallGetFromRow(row *sql.Rows) (*queuecall.Queuecall, error) {
	var referenceActiveflowID sql.NullString
	var source sql.NullString
	var tagIDs sql.NullString

	res := &queuecall.Queuecall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.QueueID,

		&res.ReferenceType,
		&res.ReferenceID,
		&referenceActiveflowID,

		&res.ForwardActionID,
		&res.ConfbridgeID,

		&source,
		&res.RoutingMethod,
		&tagIDs,

		&res.Status,
		&res.ServiceAgentID,

		&res.TimeoutWait,
		&res.TimeoutService,

		&res.DurationWaiting,
		&res.DurationService,

		&res.TMCreate,
		&res.TMService,
		&res.TMUpdate,
		&res.TMEnd,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. queueCallGetFromRow. err: %v", err)
	}

	// referenceActiveflowID
	if referenceActiveflowID.Valid {
		res.ReferenceActiveflowID = uuid.FromBytesOrNil([]byte(referenceActiveflowID.String))
	}

	// source
	if source.Valid {
		if err := json.Unmarshal([]byte(source.String), &res.Source); err != nil {
			return nil, fmt.Errorf("could not unmarshal the source. queuecallGetFromRow. err: %v", err)
		}
	}

	// tag_ids
	if tagIDs.Valid {
		if err := json.Unmarshal([]byte(tagIDs.String), &res.TagIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the source. queuecallGetFromRow. err: %v", err)
		}
	}
	if res.TagIDs == nil {
		res.TagIDs = []uuid.UUID{}
	}

	return res, nil
}

// QueuecallCreate creates new QueueCall record and returns the created QueueCall.
func (h *handler) QueuecallCreate(ctx context.Context, a *queuecall.Queuecall) error {
	q := `insert into queue_queuecalls(
		id,
		customer_id,
		queue_id,

		reference_type,
		reference_id,
		reference_activeflow_id,

		forward_action_id,
 		confbridge_id,

		source,
		routing_method,
		tag_ids,

		status,
		service_agent_id,

		timeout_wait,
		timeout_service,

		duration_waiting,
		duration_service,

		tm_create,
		tm_service,
		tm_update,
		tm_end,
		tm_delete
	) values(
		?, ?, ?,
		?, ?, ?,
		?, ?,
		?, ?, ?,
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?, ?, ?
		)
	`

	tmpSource, err := json.Marshal(a.Source)
	if err != nil {
		return fmt.Errorf("could not marshal source. QueuecallCreate. err: %v", err)
	}

	tmpTagIDs, err := json.Marshal(a.TagIDs)
	if err != nil {
		return fmt.Errorf("could not marshal tagids. QueuecallCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		a.ID.Bytes(),
		a.CustomerID.Bytes(),
		a.QueueID.Bytes(),

		a.ReferenceType,
		a.ReferenceID.Bytes(),
		a.ReferenceActiveflowID.Bytes(),

		a.ForwardActionID.Bytes(),
		a.ConfbridgeID.Bytes(),

		tmpSource,
		a.RoutingMethod,
		tmpTagIDs,

		a.Status,
		a.ServiceAgentID.Bytes(),

		a.TimeoutWait,
		a.TimeoutService,

		a.DurationWaiting,
		a.DurationService,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallCreate. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, a.ID)

	return nil
}

// queuecallUpdateToCache gets the QueueCall from the DB and update the cache.
func (h *handler) queuecallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.queuecallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.queuecallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// queuecallSetToCache sets the given queuecall to the cache
func (h *handler) queuecallSetToCache(ctx context.Context, u *queuecall.Queuecall) error {
	if err := h.cache.QueuecallSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// queuecallGetFromCache returns QueueCall from the cache.
func (h *handler) queuecallGetFromCache(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {

	// get from cache
	res, err := h.cache.QueuecallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// queuecallGetFromDB returns queuecall from the DB.
func (h *handler) queuecallGetFromDB(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", queuecallSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. queueCallGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.queuecallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. queueCallGetFromDB. err: %v", err)
	}

	return res, nil
}

// QueuecallGet get QueueCall from the database.
func (h *handler) QueuecallGet(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	res, err := h.queuecallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.queuecallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.queuecallSetToCache(ctx, res)

	return res, nil
}

// QueuecallGetByReferenceID get queuecall of the given reference id.
func (h *handler) QueuecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error) {
	tmp, err := h.cache.QueuecallGetByReferenceID(ctx, referenceID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf("%s where reference_id = ? order by tm_create desc", queuecallSelect)

	row, err := h.db.Query(q, referenceID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. QueuecallGetByReferenceID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.queuecallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. QueuecallGetByReferenceID, err: %v", err)
	}

	_ = h.queuecallSetToCache(ctx, res)

	return res, nil
}

// QueuecallGets returns queuecalls.
func (h *handler) QueuecallGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*queuecall.Queuecall, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, queuecallSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "queue_id", "reference_id", "reference_activeflow_id", "forward_action_id", "exit_action_id", "confbridge_id", "service_agent_id":
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
		return nil, fmt.Errorf("could not query. QueuecallGets. err: %v", err)
	}
	defer rows.Close()

	var res []*queuecall.Queuecall
	for rows.Next() {
		u, err := h.queuecallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. QueuecallGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// // QueuecallGetsByCustomerID returns QueueCalls.
// func (h *handler) QueuecallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*queuecall.Queuecall, error) {

// 	// prepare
// 	q := fmt.Sprintf(`%s
// 	where
// 		customer_id = ?
// 		and tm_create < ?
// 	`, queuecallSelect)

// 	values := []interface{}{
// 		customerID.Bytes(),
// 		token,
// 	}

// 	for k, v := range filters {
// 		switch k {
// 		case "deleted":
// 			if v == "false" {
// 				q = fmt.Sprintf("%s and tm_delete >= ?", q)
// 				values = append(values, DefaultTimeStamp)
// 			}

// 		case "reference_id":
// 			q = fmt.Sprintf("%s and reference_id = ?", q)
// 			tmp := uuid.FromStringOrNil(v)
// 			values = append(values, tmp.Bytes())

// 		case "queue_id":
// 			q = fmt.Sprintf("%s and queue_id = ?", q)
// 			tmp := uuid.FromStringOrNil(v)
// 			values = append(values, tmp.Bytes())

// 		case "status":
// 			q = fmt.Sprintf("%s and status = ?", q)
// 			values = append(values, v)

// 		}
// 	}

// 	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
// 	values = append(values, strconv.FormatUint(size, 10))

// 	rows, err := h.db.Query(q, values...)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not query. QueuecallGetsByCustomerID. err: %v", err)
// 	}
// 	defer rows.Close()

// 	var res []*queuecall.Queuecall
// 	for rows.Next() {
// 		u, err := h.queuecallGetFromRow(rows)
// 		if err != nil {
// 			return nil, fmt.Errorf("dbhandler: Could not scan the row. QueuecallGetsByCustomerID. err: %v", err)
// 		}

// 		res = append(res, u)
// 	}

// 	return res, nil
// }

// QueuecallDelete deletes the queuecall.
func (h *handler) QueuecallDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		queue_queuecalls
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallDelete. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusConnecting sets the QueueCall's status to the connecting.
func (h *handler) QueuecallSetStatusConnecting(ctx context.Context, id uuid.UUID, serviceAgentID uuid.UUID) error {
	// prepare
	q := `
	update
		queue_queuecalls
	set
		status = ?,
		service_agent_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queuecall.StatusConnecting, serviceAgentID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusConnecting. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusService sets the Queuecall's status to the service.
func (h *handler) QueuecallSetStatusService(ctx context.Context, id uuid.UUID, durationWaiting int, ts string) error {
	// prepare
	q := `
	update
		queue_queuecalls
	set
		status = ?,
		duration_waiting = ?,
		tm_service = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queuecall.StatusService, durationWaiting, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusService. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusAbandoned sets the Queuecall's status to the abandoned.
func (h *handler) QueuecallSetStatusAbandoned(ctx context.Context, id uuid.UUID, durationWaiting int, ts string) error {
	// prepare
	q := `
	update
		queue_queuecalls
	set
		status = ?,
		duration_waiting = ?,
		tm_end = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queuecall.StatusAbandoned, durationWaiting, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusAbandoned. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusDone sets the Queuecall's status to the done.
func (h *handler) QueuecallSetStatusDone(ctx context.Context, id uuid.UUID, durationService int, ts string) error {
	// prepare
	q := `
	update
		queue_queuecalls
	set
		status = ?,
		duration_service = ?,
		tm_end = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queuecall.StatusDone, durationService, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusDone. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusKicking sets the QueueCall's status to the kicking.
func (h *handler) QueuecallSetStatusKicking(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		queue_queuecalls
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, queuecall.StatusKicking, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusKicking. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusWaiting sets the QueueCall's status to the waiting.
func (h *handler) QueuecallSetStatusWaiting(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		queue_queuecalls
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queuecall.StatusWaiting, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusWaiting. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}
