package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-billing-manager/models/failedevent"
)

const (
	failedEventsTable = "billing_failed_events"
)

// failedEventGetFromRow gets the failed event from the row.
func (h *handler) failedEventGetFromRow(row *sql.Rows) (*failedevent.FailedEvent, error) {
	res := &failedevent.FailedEvent{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. failedEventGetFromRow. err: %v", err)
	}

	return res, nil
}

// FailedEventCreate creates a new failed event record.
func (h *handler) FailedEventCreate(ctx context.Context, c *failedevent.FailedEvent) error {
	c.TMCreate = h.utilHandler.TimeNow()
	c.TMUpdate = nil

	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("FailedEventCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(failedEventsTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("FailedEventCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("FailedEventCreate: could not execute query. err: %v", err)
	}

	return nil
}

// FailedEventListPendingRetry returns failed events that are due for retry.
func (h *handler) FailedEventListPendingRetry(ctx context.Context, now time.Time) ([]*failedevent.FailedEvent, error) {
	cols := commondatabasehandler.GetDBFields(failedevent.FailedEvent{})

	query, args, err := sq.Select(cols...).
		From(failedEventsTable).
		Where(sq.Eq{"status": []string{string(failedevent.StatusPending), string(failedevent.StatusRetrying)}}).
		Where(sq.LtOrEq{"next_retry_at": now}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("FailedEventListPendingRetry: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("FailedEventListPendingRetry: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*failedevent.FailedEvent{}
	for rows.Next() {
		fe, err := h.failedEventGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("FailedEventListPendingRetry: could not scan row. err: %v", err)
		}
		res = append(res, fe)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FailedEventListPendingRetry: row iteration error. err: %v", err)
	}

	return res, nil
}

// FailedEventUpdate updates the specified fields of a failed event.
func (h *handler) FailedEventUpdate(ctx context.Context, id uuid.UUID, fields map[failedevent.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("FailedEventUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(failedEventsTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("FailedEventUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("FailedEventUpdate: could not execute. err: %v", err)
	}

	return nil
}

// FailedEventDelete deletes a failed event record (hard delete).
func (h *handler) FailedEventDelete(ctx context.Context, id uuid.UUID) error {
	query, args, err := sq.Delete(failedEventsTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("FailedEventDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("FailedEventDelete: could not execute. err: %v", err)
	}

	return nil
}
