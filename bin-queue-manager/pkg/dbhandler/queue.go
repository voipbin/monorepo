package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-queue-manager/models/queue"
)

const (
	queueQueuesTable = "queue_queues"
)

// queueGetFromRow gets the queue from the row.
func (h *handler) queueGetFromRow(row *sql.Rows) (*queue.Queue, error) {
	res := &queue.Queue{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. queueGetFromRow. err: %v", err)
	}

	// Ensure slices are not nil
	if res.TagIDs == nil {
		res.TagIDs = []uuid.UUID{}
	}
	if res.WaitQueuecallIDs == nil {
		res.WaitQueuecallIDs = []uuid.UUID{}
	}
	if res.ServiceQueuecallIDs == nil {
		res.ServiceQueuecallIDs = []uuid.UUID{}
	}

	return res, nil
}

// QueueCreate creates new queue record and returns the created queue.
func (h *handler) QueueCreate(ctx context.Context, q *queue.Queue) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	q.TMCreate = now
	q.TMUpdate = DefaultTimeStamp
	q.TMDelete = DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(q)
	if err != nil {
		return fmt.Errorf("could not prepare fields. QueueCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(queueQueuesTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. QueueCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. QueueCreate. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, q.ID)

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
	fields := commondatabasehandler.GetDBFields(&queue.Queue{})
	query, args, err := squirrel.
		Select(fields...).
		From(queueQueuesTable).
		Where(squirrel.Eq{string(queue.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. queueGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. queueGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. queueGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.queueGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. queueGetFromDB. id: %s, err: %v", id, err)
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

// QueueList returns queues.
func (h *handler) QueueList(ctx context.Context, size uint64, token string, filters map[queue.Field]any) ([]*queue.Queue, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&queue.Queue{})
	sb := squirrel.
		Select(fields...).
		From(queueQueuesTable).
		Where(squirrel.Lt{string(queue.FieldTMCreate): token}).
		OrderBy(string(queue.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. QueueGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. QueueGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. QueueGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*queue.Queue{}
	for rows.Next() {
		u, err := h.queueGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. QueueGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. QueueGets. err: %v", err)
	}

	return res, nil
}

// QueueUpdate updates queue fields.
func (h *handler) QueueUpdate(ctx context.Context, id uuid.UUID, fields map[queue.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[queue.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("QueueUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(queueQueuesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queue.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("QueueUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("QueueUpdate: exec failed: %w", err)
	}

	_ = h.queueUpdateToCache(ctx, id)
	return nil
}

// QueueDelete deletes the queue.
func (h *handler) QueueDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[queue.Field]any{
		queue.FieldTMUpdate: ts,
		queue.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("QueueDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(queueQueuesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queue.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("QueueDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("QueueDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueAddWaitQueueCallID adds the queue call id to the queue.
// it increases the total_incoming_count + 1
func (h *handler) QueueAddWaitQueueCallID(ctx context.Context, id, queueCallID uuid.UUID) error {
	// prepare
	q := `
	update queue_queues set
		total_incoming_count = total_incoming_count + 1,
		wait_queue_call_ids = json_array_append(
			coalesce(wait_queue_call_ids,'[]'),
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueAddWaitQueueCallID. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueIncreaseTotalServicedCount increases total_serviced_count.
// It also removes the given queueCallID from the queue_call_ids.
func (h *handler) QueueIncreaseTotalServicedCount(ctx context.Context, id, queueCallID uuid.UUID) error {
	// prepare
	q := `
	update queue_queues set
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
			coalesce(service_queue_call_ids,'[]'),
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), queueCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueIncreaseTotalServicedCount. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueIncreaseTotalAbandonedCount increases total_abandoned_count.
// It also removes the given queueCallID from the queue_call_ids.
func (h *handler) QueueIncreaseTotalAbandonedCount(ctx context.Context, id, queueCallID uuid.UUID) error {
	// prepare
	q := `
	update queue_queues set
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
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueIncreaseTotalAbandonedCount. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueRemoveServiceQueueCall removes queuecall from the service_queue_call_ids.
func (h *handler) QueueRemoveServiceQueueCall(ctx context.Context, id, queueCallID uuid.UUID) error {
	// prepare
	q := `
	update queue_queues set
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
		tm_update = ?
	where
		json_search(service_queue_call_ids, 'one', ?) is not null
		and id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), h.utilHandler.TimeGetCurTime(), queueCallID.String(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueRemoveServiceQueueCall. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}

// QueueRemoveWaitQueueCall removes queuecall from the wait_queue_call_ids.
func (h *handler) QueueRemoveWaitQueueCall(ctx context.Context, id, queueCallID uuid.UUID) error {
	// prepare
	q := `
	update queue_queues set
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
		tm_update = ?
	where
		json_search(wait_queue_call_ids, 'one', ?) is not null
		and id = ?
	`

	_, err := h.db.Exec(q, queueCallID.String(), h.utilHandler.TimeGetCurTime(), queueCallID.String(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueueRemoveWaitQueueCall. err: %v", err)
	}

	// update the cache
	_ = h.queueUpdateToCache(ctx, id)

	return nil
}
