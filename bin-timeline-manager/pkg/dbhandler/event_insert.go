package dbhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventBatchInsert inserts multiple events into ClickHouse in a single batch.
// Uses PrepareBatch instead of Exec because Exec's positional parameter binding (?)
// formats time.Time with second precision (toDateTime), losing sub-second data for
// DateTime64(3) columns. PrepareBatch uses the binary columnar protocol which
// correctly preserves millisecond precision.
func (h *dbHandler) EventBatchInsert(ctx context.Context, rows []EventRow) error {
	log := logrus.WithField("func", "EventBatchInsert")

	if h.conn == nil {
		return errors.New("clickhouse connection not established")
	}

	if len(rows) == 0 {
		return nil
	}

	batch, err := h.conn.PrepareBatch(ctx, "INSERT INTO events (timestamp, event_type, publisher, data_type, data)")
	if err != nil {
		return errors.Wrap(err, "could not prepare ClickHouse batch")
	}

	for _, r := range rows {
		if err := batch.Append(r.Timestamp, r.EventType, r.Publisher, r.DataType, r.Data); err != nil {
			return errors.Wrap(err, "could not append to ClickHouse batch")
		}
	}

	if err := batch.Send(); err != nil {
		return errors.Wrap(err, "could not send ClickHouse batch")
	}

	log.Debugf("Batch inserted %d events into ClickHouse.", len(rows))
	return nil
}
