package dbhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// PeerEventBatchInsert inserts multiple peer_events rows into ClickHouse in
// a single batch. Mirrors EventBatchInsert; see that function's comment for
// why PrepareBatch (not Exec) is required for DateTime64(3) precision.
func (h *dbHandler) PeerEventBatchInsert(ctx context.Context, rows []PeerEventRow) error {
	log := logrus.WithField("func", "PeerEventBatchInsert")

	if h.conn == nil {
		return errors.New("clickhouse connection not established")
	}

	if len(rows) == 0 {
		return nil
	}

	batch, err := h.conn.PrepareBatch(ctx, "INSERT INTO peer_events (timestamp, customer_id, publisher, event_type, reference_id, direction, peer_type, peer_target, local_type, local_target, data)")
	if err != nil {
		return errors.Wrap(err, "could not prepare ClickHouse batch")
	}

	for _, r := range rows {
		if err := batch.Append(
			r.Timestamp,
			r.CustomerID,
			r.Publisher,
			r.EventType,
			r.ReferenceID,
			r.Direction,
			r.PeerType,
			r.PeerTarget,
			r.LocalType,
			r.LocalTarget,
			r.Data,
		); err != nil {
			return errors.Wrap(err, "could not append to ClickHouse batch")
		}
	}

	if err := batch.Send(); err != nil {
		return errors.Wrap(err, "could not send ClickHouse batch")
	}

	log.Debugf("Batch inserted %d peer events into ClickHouse.", len(rows))
	return nil
}
