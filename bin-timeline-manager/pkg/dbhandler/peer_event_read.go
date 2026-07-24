package dbhandler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-timeline-manager/models/peerevent"
)

// PeerPairFilter is a (peer_type, peer_target) pair used for the ClickHouse
// multi-value match. dbhandler-local only — kept out of peereventhandler's
// public interface (see pkg/peereventhandler/main.go).
type PeerPairFilter struct {
	PeerType   string
	PeerTarget string
}

// buildPeerEventQuery constructs the SQL query and args for listing peer_events.
func buildPeerEventQuery(
	customerID uuid.UUID,
	pairs []PeerPairFilter,
	pageToken string,
	pageSize int,
) (string, []interface{}) {
	query := `
		SELECT timestamp, customer_id, publisher, event_type, reference_id,
		       direction, peer_type, peer_target, local_type, local_target, data
		FROM peer_events
		WHERE customer_id = ?
	`
	args := []interface{}{customerID.String()}

	// Multi-value (peer_type, peer_target) match, OR-expanded for portability,
	// matching this package's existing buildEventConditions style (event.go
	// already prefers explicit OR-joins over driver-specific tuple-IN syntax).
	if len(pairs) > 0 {
		var ors []string
		for _, p := range pairs {
			ors = append(ors, "(peer_type = ? AND peer_target = ?)")
			args = append(args, p.PeerType, p.PeerTarget)
		}
		query += " AND (" + strings.Join(ors, " OR ") + ")"
	}

	if pageToken != "" {
		query += " AND timestamp < ?"
		args = append(args, pageToken)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, pageSize)

	return query, args
}

// PeerEventList queries peer_events from ClickHouse.
func (h *dbHandler) PeerEventList(
	ctx context.Context,
	customerID uuid.UUID,
	pairs []PeerPairFilter,
	pageToken string,
	pageSize int,
) ([]*peerevent.PeerEvent, error) {
	if h.conn == nil {
		return nil, errors.New("clickhouse connection not established")
	}

	query, args := buildPeerEventQuery(customerID, pairs, pageToken, pageSize)

	rows, err := h.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query peer_events")
	}
	defer func() { _ = rows.Close() }()

	result := []*peerevent.PeerEvent{}
	for rows.Next() {
		var e peerevent.PeerEvent
		var custIDStr, refIDStr, data string
		if err := rows.Scan(
			&e.Timestamp, &custIDStr, &e.Publisher, &e.EventType, &refIDStr,
			&e.Direction, &e.PeerType, &e.PeerTarget, &e.LocalType, &e.LocalTarget, &data,
		); err != nil {
			return nil, errors.Wrap(err, "could not scan peer_event row")
		}
		e.CustomerID = uuid.FromStringOrNil(custIDStr)
		e.ReferenceID = uuid.FromStringOrNil(refIDStr)
		e.Data = []byte(data)
		result = append(result, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	return result, nil
}
