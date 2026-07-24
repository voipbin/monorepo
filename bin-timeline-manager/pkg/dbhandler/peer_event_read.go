package dbhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-timeline-manager/models/peerevent"
)

// buildPeerEventQuery constructs the SQL query and args for listing peer_events.
// Filters on the internal-only peer_type/peer_target columns (never exposed
// by the read API) -- addrs supplies the (Type, Target) pairs to search for;
// callers pass full commonaddress.Address values but only .Type/.Target are
// used for the WHERE clause, matching the columns' actual content (see
// PeerEventRow's doc comment on why these columns exist and stay internal).
func buildPeerEventQuery(
	customerID uuid.UUID,
	addrs []commonaddress.Address,
	pageToken string,
	pageSize int,
) (string, []interface{}) {
	query := `
		SELECT timestamp, customer_id, publisher, event_type, reference_id,
		       direction, peer, local, data
		FROM peer_events
		WHERE customer_id = ?
	`
	args := []interface{}{customerID.String()}

	// Multi-value (peer_type, peer_target) match, OR-expanded for portability,
	// matching this package's existing buildEventConditions style (event.go
	// already prefers explicit OR-joins over driver-specific tuple-IN syntax).
	if len(addrs) > 0 {
		var ors []string
		for _, a := range addrs {
			ors = append(ors, "(peer_type = ? AND peer_target = ?)")
			args = append(args, string(a.Type), a.Target)
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
	addrs []commonaddress.Address,
	pageToken string,
	pageSize int,
) ([]*peerevent.PeerEvent, error) {
	if h.conn == nil {
		return nil, errors.New("clickhouse connection not established")
	}

	query, args := buildPeerEventQuery(customerID, addrs, pageToken, pageSize)

	rows, err := h.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query peer_events")
	}
	defer func() { _ = rows.Close() }()

	result := []*peerevent.PeerEvent{}
	for rows.Next() {
		var e peerevent.PeerEvent
		var custIDStr, refIDStr, data, peerJSON, localJSON string
		if err := rows.Scan(
			&e.Timestamp, &custIDStr, &e.Publisher, &e.EventType, &refIDStr,
			&e.Direction, &peerJSON, &localJSON, &data,
		); err != nil {
			return nil, errors.Wrap(err, "could not scan peer_event row")
		}
		e.CustomerID = uuid.FromStringOrNil(custIDStr)
		e.ReferenceID = uuid.FromStringOrNil(refIDStr)
		e.Data = json.RawMessage(data)

		// peer/local are stored as JSON text (see PeerEventRow's doc
		// comment); a decode failure yields a zero-value Address rather
		// than dropping the row -- the row's other fields are still valid.
		if peerJSON != "" {
			if errUnmarshal := json.Unmarshal([]byte(peerJSON), &e.Peer); errUnmarshal != nil {
				e.Peer = commonaddress.Address{}
			}
		}
		if localJSON != "" {
			if errUnmarshal := json.Unmarshal([]byte(localJSON), &e.Local); errUnmarshal != nil {
				e.Local = commonaddress.Address{}
			}
		}

		result = append(result, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	return result, nil
}
