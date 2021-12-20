package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

const (
	// select query for queue get
	queueSelect = `
	select
		id,
		user_id,
		flow_id,
		confbridge_id,
		forward_action_id,

		name,
		detail,
		webhook_uri,
		webhook_method,

		routing_method,
		tag_ids,

		wait_actions,
		wait_queue_call_ids,
		wait_timeout,
		service_queue_call_ids,
		service_timeout,

		total_incoming_count,
		total_serviced_count,
		total_abandoned_count,
		total_wait_duration,
		total_service_duration,

		tm_create,
		tm_update,
		tm_delete
	from
		queues
	`
)

// queueGetFromRow gets the queue from the row.
func (h *handler) queueGetFromRow(row *sql.Rows) (*queue.Queue, error) {

	tagIDs := ""
	waitActions := ""
	waitQueueCallIDs := ""
	serviceQueueCallIDs := ""

	res := &queue.Queue{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,
		&res.FlowID,
		&res.ConfbridgeID,
		&res.ForwardActionID,

		&res.Name,
		&res.Detail,
		&res.WebhookURI,
		&res.WebhookMethod,

		&res.RoutingMethod,
		&tagIDs,

		&waitActions,
		&waitQueueCallIDs,
		&res.WaitTimeout,
		&serviceQueueCallIDs,
		&res.ServiceTimeout,

		&res.TotalIncomingCount,
		&res.TotalServicedCount,
		&res.TotalAbandonedCount,
		&res.TotalWaitDuration,
		&res.TotalServiceDuration,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. queueGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(tagIDs), &res.TagIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the tag_ids. queueGetFromRow. err: %v", err)
	}
	if res.TagIDs == nil {
		res.TagIDs = []uuid.UUID{}
	}

	if err := json.Unmarshal([]byte(waitActions), &res.WaitActions); err != nil {
		return nil, fmt.Errorf("could not unmarshal the tag_ids. queueGetFromRow. err: %v", err)
	}
	if res.WaitActions == nil {
		res.WaitActions = []fmaction.Action{}
	}

	if err := json.Unmarshal([]byte(waitQueueCallIDs), &res.WaitQueueCallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the tag_ids. queueGetFromRow. err: %v", err)
	}
	if res.WaitQueueCallIDs == nil {
		res.WaitQueueCallIDs = []uuid.UUID{}
	}

	if err := json.Unmarshal([]byte(serviceQueueCallIDs), &res.ServiceQueueCallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the tag_ids. queueGetFromRow. err: %v", err)
	}
	if res.ServiceQueueCallIDs == nil {
		res.ServiceQueueCallIDs = []uuid.UUID{}
	}

	return res, nil
}

// QueueCreate creates new queue record and returns the created queue.
func (h *handler) QueueCreate(ctx context.Context, a *queue.Queue) error {
	q := `insert into queues(
		id,
		user_id,
		flow_id,
		confbridge_id,
		forward_action_id,

		name,
		detail,
		webhook_uri,
		webhook_method,

		routing_method,
		tag_ids,

		wait_actions,
		wait_queue_call_ids,
		wait_timeout,
		service_queue_call_ids,
		service_timeout,

		total_incoming_count,
		total_serviced_count,
		total_abandoned_count,
		total_wait_duration,
		total_service_duration,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?
		)
	`

	tagIDs, err := json.Marshal(a.TagIDs)
	if err != nil {
		return fmt.Errorf("could not marshal the tag_ids. err: %v", err)
	}
	waitActions, err := json.Marshal(a.WaitActions)
	if err != nil {
		return fmt.Errorf("could not marshal the wait_actions. err: %v", err)
	}
	waitQueueCallIDs, err := json.Marshal(a.WaitQueueCallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal the queue_call_ids. err: %v", err)
	}
	serviceQueueCallIDs, err := json.Marshal(a.ServiceQueueCallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal the queue_call_ids. err: %v", err)
	}

	_, err = h.db.Exec(q,
		a.ID.Bytes(),
		a.UserID,
		a.FlowID.Bytes(),
		a.ConfbridgeID.Bytes(),
		a.ForwardActionID.Bytes(),

		a.Name,
		a.Detail,
		a.WebhookURI,
		a.WebhookMethod,

		a.RoutingMethod,
		tagIDs,

		waitActions,
		waitQueueCallIDs,
		a.WaitTimeout,
		serviceQueueCallIDs,
		a.ServiceTimeout,

		a.TotalIncomingCount,
		a.TotalServicedCount,
		a.TotalAbandonedCount,
		a.TotalWaitDuration,
		a.TotalServiceDuration,

		a.TMCreate,
		a.TMUpdate,
		a.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute. QueueCreate. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, a.ID)

	return nil
}

// queueUpdateToCache gets the queue from the DB and update the cache.
func (h *handler) queueUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.queueGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.queueSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// queueSetToCache sets the given queue to the cache
func (h *handler) queueSetToCache(ctx context.Context, u *queue.Queue) error {
	if err := h.cache.QueueSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// queueGetFromCache returns queue from the cache.
func (h *handler) queueGetFromCache(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {

	// get from cache
	res, err := h.cache.QueueGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// queueGetFromDB returns queue from the DB.
func (h *handler) queueGetFromDB(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", queueSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. queueGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.queueGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. queueGetFromDB. err: %v", err)
	}

	return res, nil
}

// QueueGet get queue from the database.
func (h *handler) QueueGet(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	res, err := h.queueGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.queueGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.queueSetToCache(ctx, res)

	return res, nil
}

// QueueGets returns queues.
func (h *handler) QueueGets(ctx context.Context, userID uint64, size uint64, token string) ([]*queue.Queue, error) {
	// prepare
	q := fmt.Sprintf("%s where user_id = ? and tm_create < ? order by tm_create desc limit ?", queueSelect)

	rows, err := h.db.Query(q, userID, token, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. QueueGets. err: %v", err)
	}
	defer rows.Close()

	var res []*queue.Queue
	for rows.Next() {
		u, err := h.queueGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. QueueGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// QueueDelete deletes the queue.
func (h *handler) QueueDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		queues
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	t := GetCurTime()
	_, err := h.db.Exec(q, t, t, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueDelete. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueSetBasicInfo sets the queue's basic info.
func (h *handler) QueueSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail, webhookURI, webhookMethod string) error {
	// prepare
	q := `
	update
		queues
	set
		name = ?,
		detail = ?,
		webhook_uri = ?,
		webhook_method = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, name, detail, webhookURI, webhookMethod, GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueSetBasicInfo. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueSetRoutingMethod sets the queue's routing_method.
func (h *handler) QueueSetRoutingMethod(ctx context.Context, id uuid.UUID, routingMethod queue.RoutingMethod) error {
	// prepare
	q := `
	update
		queues
	set
		routing_method = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, routingMethod, GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueSetRoutingMethod. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueSetTagIDs sets the queue's tag_ids.
func (h *handler) QueueSetTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) error {
	// prepare
	q := `
	update
		queues
	set
		tag_ids = ?,
		tm_update = ?
	where
		id = ?
	`

	t, err := json.Marshal(tagIDs)
	if err != nil {
		return fmt.Errorf("could not marshal the tag_ids. err: %v", err)
	}

	_, err = h.db.Exec(q, t, GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueSetTagIDs. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueSetWaitActionsAndTimeouts sets the queue's wait_actions.
func (h *handler) QueueSetWaitActionsAndTimeouts(ctx context.Context, id uuid.UUID, waitActions []fmaction.Action, waitTimeout, serviceTimeout int) error {
	// prepare
	q := `
	update
		queues
	set
		wait_actions = ?,
		wait_timeout = ?,
		service_timeout = ?,
		tm_update = ?
	where
		id = ?
	`

	t, err := json.Marshal(waitActions)
	if err != nil {
		return fmt.Errorf("could not marshal the tag_ids. err: %v", err)
	}

	_, err = h.db.Exec(q, t, waitTimeout, serviceTimeout, GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueSetWaitActionsAndTimeouts. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueAddQueueCallID adds the queue call id to the queue.
// it increases the total_incoming_count + 1
func (h *handler) QueueAddQueueCallID(ctx context.Context, id, queueCallID uuid.UUID) error {
	// prepare
	q := `
	update queues set
		total_incoming_count = total_incoming_count + 1,
		wait_queue_call_ids = json_array_append(
			wait_queue_call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueAddQueueCallID. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueIncreaseTotalServicedCount increases total_serviced_count.
// It also removes the given queueCallID from the queue_call_ids.
// It also add the waittime to the total_wait_duration.
func (h *handler) QueueIncreaseTotalServicedCount(ctx context.Context, id, queueCallID uuid.UUID, waittime time.Duration) error {
	// prepare
	q := `
	update queues set
		total_serviced_count = total_serviced_count + 1,
		wait_queue_call_ids = json_remove(
			wait_queue_call_ids, replace(
				json_search(
					wait_queue_call_ids,
					'one',
					?
				),
				'"',
				''
			)
		),
		service_queue_call_ids = json_array_append(
			service_queue_call_ids,
			'$',
			?
		),
		total_wait_duration = total_wait_duration + ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), queueCallID.String(), waittime.Milliseconds(), GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueIncreaseTotalServicedCount. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueIncreaseTotalAbandonedCount increases total_abandoned_count.
// It also removes the given queueCallID from the queue_call_ids.
// It also add the waittime to the total_waittime.
func (h *handler) QueueIncreaseTotalAbandonedCount(ctx context.Context, id, queueCallID uuid.UUID, waittime time.Duration) error {
	// prepare
	q := `
	update queues set
		total_abandoned_count = total_abandoned_count + 1,
		wait_queue_call_ids = json_remove(
			wait_queue_call_ids, replace(
				json_search(
					wait_queue_call_ids,
					'one',
					?
				),
				'"',
				''
			)
		),
		total_wait_duration = total_wait_duration + ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), waittime.Milliseconds(), GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueIncreaseTotalAbandonedCount. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueRemoveServiceQueueCall removes queuecall from the service_queue_call_ids.
// It also add the serviceTime to the total_service_duration.
func (h *handler) QueueRemoveServiceQueueCall(ctx context.Context, id, queueCallID uuid.UUID, serviceTime time.Duration) error {
	// prepare
	q := `
	update queues set
		total_serviced_count = total_serviced_count + 1,
		service_queue_call_ids = json_remove(
			service_queue_call_ids, replace(
				json_search(
					service_queue_call_ids,
					'one',
					?
				),
				'"',
				''
			)
		),
		total_service_duration = total_service_duration + ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), serviceTime.Milliseconds(), GetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueRemoveServiceQueueCall. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}
