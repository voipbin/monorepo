package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

const (
	// select query for  queuecall get
	queueCallSelect = `
	select
		id,
		customer_id,
		queue_id,

		reference_type,
		reference_id,
		reference_activeflow_id,

		flow_id,
		forward_action_id,
		exit_action_id,
		confbridge_id,

		source,
		routing_method,
		tag_ids,

		status,
		service_agent_id,

		timeout_wait,
		timeout_service,

		tm_create,
		tm_service,
		tm_update,
		tm_delete
	from
		queuecalls
	`
)

// queuecallGetFromRow gets the  queuecall from the row.
func (h *handler) queuecallGetFromRow(row *sql.Rows) (*queuecall.Queuecall, error) {

	source := ""
	tagIDs := ""

	res := &queuecall.Queuecall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.QueueID,

		&res.ReferenceType,
		&res.ReferenceID,
		&res.ReferenceActiveflowID,

		&res.FlowID,
		&res.ForwardActionID,
		&res.ExitActionID,
		&res.ConfbridgeID,

		&source,
		&res.RoutingMethod,
		&tagIDs,

		&res.Status,
		&res.ServiceAgentID,

		&res.TimeoutWait,
		&res.TimeoutService,

		&res.TMCreate,
		&res.TMService,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. queueCallGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(source), &res.Source); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. queuecallGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(tagIDs), &res.TagIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the tag_ids. queuecallGetFromRow. err: %v", err)
	}
	if res.TagIDs == nil {
		res.TagIDs = []uuid.UUID{}
	}

	return res, nil
}

// QueuecallCreate creates new QueueCall record and returns the created QueueCall.
func (h *handler) QueuecallCreate(ctx context.Context, a *queuecall.Queuecall) error {
	q := `insert into queuecalls(
		id,
		customer_id,
		queue_id,

		reference_type,
		reference_id,
		reference_activeflow_id,

		flow_id,
		forward_action_id,
		exit_action_id,
		confbridge_id,

		source,
		routing_method,
		tag_ids,

		status,
		service_agent_id,

		timeout_wait,
		timeout_service,

		tm_create,
		tm_service,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?,
		?, ?,
		?, ?,
		?, ?, ?, ?
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

		a.FlowID.Bytes(),
		a.ForwardActionID.Bytes(),
		a.ExitActionID.Bytes(),
		a.ConfbridgeID.Bytes(),

		tmpSource,
		a.RoutingMethod,
		tmpTagIDs,

		a.Status,
		a.ServiceAgentID.Bytes(),

		a.TimeoutWait,
		a.TimeoutService,

		a.TMCreate,
		a.TMService,
		a.TMUpdate,
		a.TMDelete,
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
	q := fmt.Sprintf("%s where id = ?", queueCallSelect)

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

// QueuecallGets returns QueueCalls.
func (h *handler) QueuecallGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*queuecall.Queuecall, error) {
	// prepare
	q := fmt.Sprintf("%s where customer_id = ? and tm_create < ? order by tm_create desc limit ?", queueCallSelect)

	rows, err := h.db.Query(q, customerID.Bytes(), token, size)
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

// QueuecallGetsByReferenceID returns QueueCalls.
func (h *handler) QueuecallGetsByReferenceID(ctx context.Context, referenceID uuid.UUID) ([]*queuecall.Queuecall, error) {
	// prepare
	q := fmt.Sprintf("%s where reference_id = ?", queueCallSelect)

	rows, err := h.db.Query(q, referenceID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. QueuecallGetsByReferenceID. err: %v", err)
	}
	defer rows.Close()

	var res []*queuecall.Queuecall
	for rows.Next() {
		u, err := h.queuecallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. QueuecallGetsByReferenceID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// QueuecallDelete deletes the queuecall.
func (h *handler) QueuecallDelete(ctx context.Context, id uuid.UUID, status queuecall.Status) error {
	// prepare
	q := `
	update
		queuecalls
	set
		status = ?,
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	t := GetCurTime()
	_, err := h.db.Exec(q, status, t, t, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallDelete. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetServiceAgentID sets the QueueCall's service_agent_id.
func (h *handler) QueuecallSetServiceAgentID(ctx context.Context, id uuid.UUID, serviceAgentID uuid.UUID) error {
	// prepare
	q := `
	update
		queuecalls
	set
		status = ?,
		service_agent_id = ?,
		tm_service = ?,
		tm_update = ?
	where
		id = ?
	`
	t := GetCurTime()
	_, err := h.db.Exec(q, queuecall.StatusEntering, serviceAgentID.Bytes(), t, t, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetServiceAgentID. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusService sets the QueueCall's status to the service.
func (h *handler) QueuecallSetStatusService(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		queuecalls
	set
		status = ?,
		tm_service = ?,
		tm_update = ?
	where
		id = ?
	`
	t := GetCurTime()
	_, err := h.db.Exec(q, queuecall.StatusService, t, t, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusService. err: %v", err)
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
		queuecalls
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, queuecall.StatusKicking, GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusKicking. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil

}
