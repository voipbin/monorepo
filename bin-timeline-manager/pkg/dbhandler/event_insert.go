package dbhandler

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventInsert inserts a single event into ClickHouse.
// Uses PrepareBatch instead of Exec because Exec's positional parameter binding (?)
// formats time.Time with second precision (toDateTime), losing sub-second data for
// DateTime64(3) columns. PrepareBatch uses the binary columnar protocol which
// correctly preserves millisecond precision.
func (h *dbHandler) EventInsert(ctx context.Context, timestamp time.Time, eventType string, publisher string, dataType string, data string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "EventInsert",
		"event_type": eventType,
		"publisher":  publisher,
	})

	if h.conn == nil {
		return errors.New("clickhouse connection not established")
	}

	batch, err := h.conn.PrepareBatch(ctx, "INSERT INTO events (timestamp, event_type, publisher, data_type, data)")
	if err != nil {
		return errors.Wrap(err, "could not prepare ClickHouse batch")
	}

	if err := batch.Append(timestamp, eventType, publisher, dataType, data); err != nil {
		return errors.Wrap(err, "could not append to ClickHouse batch")
	}

	if err := batch.Send(); err != nil {
		return errors.Wrap(err, "could not send ClickHouse batch")
	}

	log.Debugf("Inserted event into ClickHouse. event_type: %s, publisher: %s", eventType, publisher)
	return nil
}
