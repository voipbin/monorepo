package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

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
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	qc.TMCreate = now
	qc.TMService = DefaultTimeStamp
	qc.TMUpdate = DefaultTimeStamp
	qc.TMEnd = DefaultTimeStamp
	qc.TMDelete = DefaultTimeStamp

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

// QueuecallGets returns queuecalls.
func (h *handler) QueuecallGets(ctx context.Context, size uint64, token string, filters map[queuecall.Field]any) ([]*queuecall.Queuecall, error) {
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

// QueuecallUpdate updates queuecall fields.
func (h *handler) QueuecallUpdate(ctx context.Context, id uuid.UUID, fields map[queuecall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[queuecall.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

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
	ts := h.utilHandler.TimeGetCurTime()

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
func (h *handler) QueuecallSetStatusConnecting(ctx context.Context, id uuid.UUID, serviceAgentID uuid.UUID) error {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus:         queuecall.StatusConnecting,
		queuecall.FieldServiceAgentID: serviceAgentID,
	}

	if err := h.QueuecallUpdate(ctx, id, fields); err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusConnecting. err: %v", err)
	}

	return nil
}

// QueuecallSetStatusService sets the Queuecall's status to the service.
func (h *handler) QueuecallSetStatusService(ctx context.Context, id uuid.UUID, durationWaiting int, ts string) error {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus:          queuecall.StatusService,
		queuecall.FieldDurationWaiting: durationWaiting,
		queuecall.FieldTMService:       ts,
		queuecall.FieldTMUpdate:        ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("QueuecallSetStatusService: prepare fields failed: %w", err)
	}

	q := squirrel.Update(queueQueuecallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("QueuecallSetStatusService: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("QueuecallSetStatusService: exec failed: %w", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusAbandoned sets the Queuecall's status to the abandoned.
func (h *handler) QueuecallSetStatusAbandoned(ctx context.Context, id uuid.UUID, durationWaiting int, ts string) error {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus:          queuecall.StatusAbandoned,
		queuecall.FieldDurationWaiting: durationWaiting,
		queuecall.FieldTMEnd:           ts,
		queuecall.FieldTMUpdate:        ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("QueuecallSetStatusAbandoned: prepare fields failed: %w", err)
	}

	q := squirrel.Update(queueQueuecallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("QueuecallSetStatusAbandoned: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("QueuecallSetStatusAbandoned: exec failed: %w", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
}

// QueuecallSetStatusDone sets the Queuecall's status to the done.
func (h *handler) QueuecallSetStatusDone(ctx context.Context, id uuid.UUID, durationService int, ts string) error {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus:          queuecall.StatusDone,
		queuecall.FieldDurationService: durationService,
		queuecall.FieldTMEnd:           ts,
		queuecall.FieldTMUpdate:        ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("QueuecallSetStatusDone: prepare fields failed: %w", err)
	}

	q := squirrel.Update(queueQueuecallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(queuecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("QueuecallSetStatusDone: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("QueuecallSetStatusDone: exec failed: %w", err)
	}

	// update the cache
	_ = h.queuecallUpdateToCache(ctx, id)

	return nil
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

// QueuecallSetStatusWaiting sets the QueueCall's status to the waiting.
func (h *handler) QueuecallSetStatusWaiting(ctx context.Context, id uuid.UUID) error {
	fields := map[queuecall.Field]any{
		queuecall.FieldStatus: queuecall.StatusWaiting,
	}

	if err := h.QueuecallUpdate(ctx, id, fields); err != nil {
		return fmt.Errorf("could not execute. QueuecallSetStatusWaiting. err: %v", err)
	}

	return nil
}
