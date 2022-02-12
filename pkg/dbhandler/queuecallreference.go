package dbhandler

import (
	context "context"
	"database/sql"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
)

const (
	// select query for QueuecallReference get
	queuecallReferenceSelect = `
	select
		id,
		customer_id,
		type,

		current_queuecall_id,
		queuecall_ids,

		tm_create,
		tm_update,
		tm_delete
	from
		queuecallreferences
	`
)

// queuecallReferenceGetFromRow gets the  queuecall from the row.
func (h *handler) queuecallReferenceGetFromRow(row *sql.Rows) (*queuecallreference.QueuecallReference, error) {
	queuecallIDs := ""

	res := &queuecallreference.QueuecallReference{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.Type,

		&res.CurrentQueuecallID,
		&queuecallIDs,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. queuecallReferenceGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(queuecallIDs), &res.QueuecallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. queuecallReferenceGetFromRow. err: %v", err)
	}
	if res.QueuecallIDs == nil {
		res.QueuecallIDs = []uuid.UUID{}
	}

	return res, nil
}

// QueuecallReferenceCreate creates new QueuecallReference record.
func (h *handler) QueuecallReferenceCreate(ctx context.Context, a *queuecallreference.QueuecallReference) error {
	q := `insert into queuecallreferences(
		id,
		customer_id,
		type,

		current_queuecall_id,
		queuecall_ids,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?, ?, ?
	)
	`

	tmpQueuecallIDs, err := json.Marshal(a.QueuecallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal source. QueuecallReferenceCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		a.ID.Bytes(),
		a.CustomerID.Bytes(),
		a.Type,

		a.CurrentQueuecallID.Bytes(),
		tmpQueuecallIDs,

		a.TMCreate,
		a.TMUpdate,
		a.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallReferenceCreate. err: %v", err)
	}

	// update the cache
	_ = h.queuecallReferenceUpdateToCache(ctx, a.ID)

	return nil
}

// queuecallReferenceUpdateToCache gets the QueuecallReference from the DB and update the cache.
func (h *handler) queuecallReferenceUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.queuecallReferenceGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.queuecallReferenceSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// queuecallReferenceSetToCache sets the given queuecall to the cache
func (h *handler) queuecallReferenceSetToCache(ctx context.Context, u *queuecallreference.QueuecallReference) error {
	if err := h.cache.QueuecallReferenceSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// queuecallReferenceGetFromCache returns QueuecallReference from the cache.
func (h *handler) queuecallReferenceGetFromCache(ctx context.Context, id uuid.UUID) (*queuecallreference.QueuecallReference, error) {

	// get from cache
	res, err := h.cache.QueuecallReferenceGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// queuecallReferenceGetFromDB returns queuecall from the DB.
func (h *handler) queuecallReferenceGetFromDB(ctx context.Context, id uuid.UUID) (*queuecallreference.QueuecallReference, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", queuecallReferenceSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. queuecallReferenceGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.queuecallReferenceGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. queuecallReferenceGetFromDB. err: %v", err)
	}

	return res, nil
}

// QueuecallReferenceGet get QueueCall from the database.
func (h *handler) QueuecallReferenceGet(ctx context.Context, id uuid.UUID) (*queuecallreference.QueuecallReference, error) {

	res, err := h.queuecallReferenceGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.queuecallReferenceGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.queuecallReferenceSetToCache(ctx, res)

	return res, nil
}

// QueuecallReferenceSetCurrentQueuecallID sets the given queuecallID to the current_queuecall_id.
func (h *handler) QueuecallReferenceSetCurrentQueuecallID(ctx context.Context, id, queuecallID uuid.UUID) error {

	// prepare
	q := `
	update queuecallreferences set
		current_queuecall_id = ?,
		queuecall_ids = json_array_append(
			queuecall_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	t := GetCurTime()
	_, err := h.db.Exec(q, queuecallID.Bytes(), queuecallID.String(), t, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallReferenceSetCurrentQueuecallID. err: %v", err)
	}

	// update the cache
	_ = h.queuecallReferenceUpdateToCache(ctx, id)

	return nil
}

// QueuecallReferenceDelete deletes the queuecallreference.
func (h *handler) QueuecallReferenceDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		queuecallreferences
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	t := GetCurTime()
	_, err := h.db.Exec(q, t, t, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. QueuecallReferenceDelete. err: %v", err)
	}

	// update the cache
	_ = h.queuecallReferenceUpdateToCache(ctx, id)

	return nil
}
