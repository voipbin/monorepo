package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-queue-manager/models/queuecall"
)

const (
	queueQueuecallsTable = "queue_queuecalls"
)

// queuecallGetFromRow gets the queuecall from the row.
func (h *handler) queuecallGetFromRow(row *sql.Rows) (*queuecall.Queuecall, error) {
	res := &queuecall.Queuecall{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. queuecallGetFromRow. err: %v", err)
	}

	// Ensure Source is not nil
	if res.Source == (commonaddress.Address{}) {
		res.Source = commonaddress.Address{}
	}

	// Ensure TagIDs slice is not nil
	if res.TagIDs == nil {
		res.TagIDs = []uuid.UUID{}
	}

	return res, nil
}

// QueuecallCreate creates new QueueCall record and returns the created QueueCall.
func (h *handler) QueuecallCreate(ctx context.Context, qc *queuecall.Queuecall) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	qc.TMCreate = now
	qc.TMService = nil
	qc.TMUpdate = nil
	qc.TMEnd = nil
	qc.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(qc)
	if err != nil {
		return fmt.Errorf("could not prepare fields. QueuecallCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(queueQueuecallsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. QueuecallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. QueuecallCreate. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, qc.ID)

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
	fields := commondatabasehandler.GetDBFields(&queuecall.Queuecall{})
	query, args, err := squirrel.
		Select(fields...).
		From(queueQueuecallsTable).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. queuecallGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. queuecallGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. queuecallGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.queuecallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. queuecallGetFromDB. id: %s, err: %v", id, err)
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

	fields := commondatabasehandler.GetDBFields(&queuecall.Queuecall{})
	query, args, err := squirrel.
		Select(fields...).
		From(queueQueuecallsTable).
		Where(squirrel.Eq{string(queuecall.FieldReferenceID): referenceID.Bytes()}).
		OrderBy(string(queuecall.FieldTMCreate) + " DESC").
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. QueuecallGetByReferenceID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. QueuecallGetByReferenceID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.queuecallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get queuecall. QueuecallGetByReferenceID, err: %v", err)
	}

	_ = h.queuecallSetToCache(ctx, res)

	return res, nil
}

// QueuecallList returns queuecalls.
func (h *handler) QueuecallList(ctx context.Context, size uint64, token string, filters map[queuecall.Field]any) ([]*queuecall.Queuecall, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&queuecall.Queuecall{})
	sb := squirrel.
		Select(fields...).
		From(queueQueuecallsTable).
		Where(squirrel.Lt{string(queuecall.FieldTMCreate): token}).
		OrderBy(string(queuecall.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. QueuecallGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. QueuecallGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. QueuecallGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*queuecall.Queuecall{}
	for rows.Next() {
		u, err := h.queuecallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. QueuecallGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. QueuecallGets. err: %v", err)
	}

	return res, nil
}

// QueuecallListOldestWaiting returns up to limit waiting queuecalls for the
// given queue, ordered oldest-first (VOIP-1539 §3.5, event entry point B: the
// FIFO pick used when an agent becomes available). Deliberately separate from
// QueuecallList's DESC/tm_create-token pagination contract -- this is a
// small, fixed-direction, non-paginated query.
func (h *handler) QueuecallListOldestWaiting(ctx context.Context, queueID uuid.UUID, limit uint64) ([]*queuecall.Queuecall, error) {
	fields := commondatabasehandler.GetDBFields(&queuecall.Queuecall{})
	sb := squirrel.
		Select(fields...).
		From(queueQueuecallsTable).
		Where(squirrel.Eq{
			string(queuecall.FieldQueueID): queueID.Bytes(),
			string(queuecall.FieldStatus):  string(queuecall.StatusWaiting),
		}).
		OrderBy(string(queuecall.FieldTMCreate) + " ASC").
		Limit(limit).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. QueuecallListOldestWaiting. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. QueuecallListOldestWaiting. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*queuecall.Queuecall{}
	for rows.Next() {
		u, err := h.queuecallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. QueuecallListOldestWaiting, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. QueuecallListOldestWaiting. err: %v", err)
	}

	return res, nil
}

// QueuecallListConnectingStale returns up to limit queuecalls stuck in the
// connecting status whose tm_update is older than before, ordered
// oldest-stale-first (VOIP-1539 §5.2, matching backstop recovery A:
// connecting-stale rollback).
func (h *handler) QueuecallListConnectingStale(ctx context.Context, before time.Time, limit uint64) ([]*queuecall.Queuecall, error) {
	fields := commondatabasehandler.GetDBFields(&queuecall.Queuecall{})
	sb := squirrel.
		Select(fields...).
		From(queueQueuecallsTable).
		Where(squirrel.Eq{
			string(queuecall.FieldStatus): string(queuecall.StatusConnecting),
		}).
		Where(squirrel.Lt{
			string(queuecall.FieldTMUpdate): before,
		}).
		OrderBy(string(queuecall.FieldTMUpdate) + " ASC").
		Limit(limit).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. QueuecallListConnectingStale. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. QueuecallListConnectingStale. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*queuecall.Queuecall{}
	for rows.Next() {
		u, err := h.queuecallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. QueuecallListConnectingStale, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. QueuecallListConnectingStale. err: %v", err)
	}

	return res, nil
}

// QueuecallListWaitingOldest returns up to limit waiting queuecalls across
// all queues, ordered oldest-first (VOIP-1539 §5.2, matching backstop
// recovery B: waiting queuecalls get a fresh match() attempt). Unlike
// QueuecallListOldestWaiting this is not scoped to a single queue -- the
// backstop sweeps every waiting queuecall in one pass.
func (h *handler) QueuecallListWaitingOldest(ctx context.Context, limit uint64) ([]*queuecall.Queuecall, error) {
	fields := commondatabasehandler.GetDBFields(&queuecall.Queuecall{})
	sb := squirrel.
		Select(fields...).
		From(queueQueuecallsTable).
		Where(squirrel.Eq{
			string(queuecall.FieldStatus): string(queuecall.StatusWaiting),
		}).
		OrderBy(string(queuecall.FieldTMCreate) + " ASC").
		Limit(limit).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. QueuecallListWaitingOldest. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. QueuecallListWaitingOldest. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*queuecall.Queuecall{}
	for rows.Next() {
		u, err := h.queuecallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. QueuecallListWaitingOldest, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. QueuecallListWaitingOldest. err: %v", err)
	}

	return res, nil
}

// QueuecallUpdate updates queuecall fields.
func (h *handler) QueuecallUpdate(ctx context.Context, id uuid.UUID, fields map[queuecall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[queuecall.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("QueuecallUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(queueQueuecallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("QueuecallUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("QueuecallUpdate: exec failed: %w", err)
	}

	_ = h.queuecallUpdateToCache(ctx, id)
	return nil
}

// QueuecallDelete deletes the queuecall.
func (h *handler) QueuecallDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[queuecall.Field]any{
		queuecall.FieldTMUpdate: ts,
		queuecall.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("QueuecallDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(queueQueuecallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("QueuecallDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("QueuecallDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusConnecting sets the QueueCall's status to the connecting.
//
// It performs a compare-and-swap UPDATE guarded by `status = 'waiting'`
// (VOIP-1539 §5.0). Only a waiting queuecall can transition to connecting. It
// returns the number of affected rows: 1 when this call won the CAS, 0 when the
// queuecall was not in the waiting status (already connecting/serviced/ended by
// a racing writer).
func (h *handler) QueuecallSetStatusConnecting(ctx context.Context, id uuid.UUID, serviceAgentID uuid.UUID, groupcallID uuid.UUID) (int64, error) {
	fields, err := commondatabasehandler.PrepareFields(map[queuecall.Field]any{
		queuecall.FieldStatus:         queuecall.StatusConnecting,
		queuecall.FieldServiceAgentID: serviceAgentID,
		queuecall.FieldGroupcallID:    groupcallID,
		queuecall.FieldTMUpdate:       h.utilHandler.TimeNow(),
	})
	if err != nil {
		return 0, fmt.Errorf("could not prepare fields. QueuecallSetStatusConnecting. err: %v", err)
	}

	sqlStr, args, err := squirrel.
		Update(queueQueuecallsTable).
		SetMap(fields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		Where(squirrel.Eq{string(queuecall.FieldStatus): string(queuecall.StatusWaiting)}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query. QueuecallSetStatusConnecting. err: %v", err)
	}

	res, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("could not execute query. QueuecallSetStatusConnecting. err: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("could not get rows affected. QueuecallSetStatusConnecting. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return affected, nil
}

// QueuecallSetStatusService sets the Queuecall's status to the service.
//
// It performs a compare-and-swap UPDATE guarded by `status = 'connecting'`
// (VOIP-1539 §5.0). Only a connecting queuecall can transition to service. It
// returns the number of affected rows: 1 when this call won the CAS, 0 when the
// queuecall was not in the connecting status (already serviced/abandoned/rolled
// back by a racing writer).
func (h *handler) QueuecallSetStatusService(ctx context.Context, id uuid.UUID, durationWaiting int, ts *time.Time) (int64, error) {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus:          queuecall.StatusService,
		queuecall.FieldDurationWaiting: durationWaiting,
		queuecall.FieldTMService:       ts,
		queuecall.FieldTMUpdate:        ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusService: prepare fields failed: %w", err)
	}

	q := squirrel.Update(queueQueuecallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		Where(squirrel.Eq{string(queuecall.FieldStatus): string(queuecall.StatusConnecting)}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusService: build SQL failed: %w", err)
	}

	res, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusService: exec failed: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusService: rows affected failed: %w", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return affected, nil
}

// QueuecallSetStatusAbandoned sets the Queuecall's status to the abandoned.
//
// It performs a compare-and-swap UPDATE guarded by
// `tm_end IS NULL AND status != 'service'` (VOIP-1539 §5.0). A queuecall can be
// abandoned from any non-serviced, not-yet-ended status. It returns the number
// of affected rows: 1 when this call won the CAS, 0 when the queuecall was
// already serviced or ended by a racing writer (e.g. join won and it is now in
// service, so its confbridge must not be torn down).
func (h *handler) QueuecallSetStatusAbandoned(ctx context.Context, id uuid.UUID, durationWaiting int, ts *time.Time) (int64, error) {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus:          queuecall.StatusAbandoned,
		queuecall.FieldDurationWaiting: durationWaiting,
		queuecall.FieldTMEnd:           ts,
		queuecall.FieldTMUpdate:        ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusAbandoned: prepare fields failed: %w", err)
	}

	q := squirrel.Update(queueQueuecallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		Where(squirrel.Eq{string(queuecall.FieldTMEnd): nil}).
		Where(squirrel.NotEq{string(queuecall.FieldStatus): string(queuecall.StatusService)}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusAbandoned: build SQL failed: %w", err)
	}

	res, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusAbandoned: exec failed: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusAbandoned: rows affected failed: %w", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return affected, nil
}

// QueuecallSetStatusDone sets the Queuecall's status to the done.
//
// It performs a compare-and-swap UPDATE guarded by `status = 'service'`
// (VOIP-1539 §5.0). Only a serviced queuecall can transition to done. It returns
// the number of affected rows: 1 when this call won the CAS, 0 when the
// queuecall was not in the service status (already done/abandoned by a racing
// writer), so its confbridge is not deleted twice.
func (h *handler) QueuecallSetStatusDone(ctx context.Context, id uuid.UUID, durationService int, ts *time.Time) (int64, error) {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus:          queuecall.StatusDone,
		queuecall.FieldDurationService: durationService,
		queuecall.FieldTMEnd:           ts,
		queuecall.FieldTMUpdate:        ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusDone: prepare fields failed: %w", err)
	}

	q := squirrel.Update(queueQueuecallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		Where(squirrel.Eq{string(queuecall.FieldStatus): string(queuecall.StatusService)}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusDone: build SQL failed: %w", err)
	}

	res, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusDone: exec failed: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("QueuecallSetStatusDone: rows affected failed: %w", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return affected, nil
}

// QueuecallSetStatusKicking sets the QueueCall's status to the kicking.
func (h *handler) QueuecallSetStatusKicking(ctx context.Context, id uuid.UUID) error {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus: queuecall.StatusKicking,
	}

	if err := h.QueuecallUpdate(ctx, id, fields); err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusKicking. err: %v", err)
	}

	return nil
}

// QueuecallSetStatusWaitingIfInitiating sets the QueueCall's status to the
// waiting, guarded by `status = 'initiating'` (VOIP-1539 §5.0 #1).
//
// It performs a compare-and-swap UPDATE: only an initiating queuecall can
// transition to waiting (the normal enqueue path). It returns the number of
// affected rows: 1 when this call won the CAS, 0 when the queuecall was not in
// the initiating status (e.g. already abandoned by a racing customer hangup, so
// a blind waiting-write must not resurrect the abandoned call).
func (h *handler) QueuecallSetStatusWaitingIfInitiating(ctx context.Context, id uuid.UUID) (int64, error) {
	return h.queuecallSetStatusWaitingIf(ctx, id, queuecall.StatusInitiating)
}

// QueuecallSetStatusWaitingIfConnecting sets the QueueCall's status to the
// waiting, guarded by `status = 'connecting'` (VOIP-1539 §5.0 #2).
//
// It performs a compare-and-swap UPDATE: only a connecting queuecall can be
// rolled back to waiting (the connecting-stale backstop rollback path). It
// returns the number of affected rows: 1 when this call won the CAS, 0 when the
// queuecall was not in the connecting status (e.g. join won and it is now in
// service, so the rollback must be a no-op).
func (h *handler) QueuecallSetStatusWaitingIfConnecting(ctx context.Context, id uuid.UUID) (int64, error) {
	return h.queuecallSetStatusWaitingIf(ctx, id, queuecall.StatusConnecting)
}

// queuecallSetStatusWaitingIf performs a compare-and-swap UPDATE that sets the
// queuecall's status to waiting, guarded by the given expected current status.
func (h *handler) queuecallSetStatusWaitingIf(ctx context.Context, id uuid.UUID, expectStatus queuecall.Status) (int64, error) {
	fields, err := commondatabasehandler.PrepareFields(map[queuecall.Field]any{
		queuecall.FieldStatus:   queuecall.StatusWaiting,
		queuecall.FieldTMUpdate: h.utilHandler.TimeNow(),
	})
	if err != nil {
		return 0, fmt.Errorf("could not prepare fields. queuecallSetStatusWaitingIf. err: %v", err)
	}

	sqlStr, args, err := squirrel.
		Update(queueQueuecallsTable).
		SetMap(fields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		Where(squirrel.Eq{string(queuecall.FieldStatus): string(expectStatus)}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query. queuecallSetStatusWaitingIf. err: %v", err)
	}

	res, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("could not execute query. queuecallSetStatusWaitingIf. err: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("could not get rows affected. queuecallSetStatusWaitingIf. err: %v", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return affected, nil
}
