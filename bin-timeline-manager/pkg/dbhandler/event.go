package dbhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-timeline-manager/models/event"
)

// buildEventQuery constructs the SQL query and args for listing events.
func buildEventQuery(
	publisher commonoutline.ServiceName,
	resourceID uuid.UUID,
	events []string,
	pageToken string,
	pageSize int,
) (string, []interface{}) {
	query := `
		SELECT timestamp, event_type, publisher, data_type, data
		FROM events
		WHERE publisher = ?
		  AND resource_id = ?
	`
	args := []interface{}{string(publisher), resourceID.String()}

	// Add event type filters
	if len(events) > 0 {
		conditions := buildEventConditions(events)
		if conditions != "" {
			query += " AND (" + conditions + ")"
		}
	}

	// Pagination by timestamp
	if pageToken != "" {
		query += " AND timestamp < ?"
		args = append(args, pageToken)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, pageSize)

	return query, args
}

// EventList queries events from ClickHouse.
func (h *dbHandler) EventList(
	ctx context.Context,
	publisher commonoutline.ServiceName,
	resourceID uuid.UUID,
	events []string,
	pageToken string,
	pageSize int,
) ([]*event.Event, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventList",
		"publisher":   publisher,
		"resource_id": resourceID,
		"events":      events,
		"page_token":  pageToken,
		"page_size":   pageSize,
	})

	if h.conn == nil {
		return nil, errors.New("clickhouse connection not established")
	}

	query, args := buildEventQuery(publisher, resourceID, events, pageToken, pageSize)

	log.Debugf("Executing query: %s with args: %v", query, args)

	rows, err := h.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query events")
	}
	defer func() { _ = rows.Close() }()

	var result []*event.Event
	for rows.Next() {
		var e event.Event
		var data string
		if err := rows.Scan(&e.Timestamp, &e.EventType, &e.Publisher, &e.DataType, &data); err != nil {
			return nil, errors.Wrap(err, "could not scan event row")
		}
		e.Data = []byte(data)
		result = append(result, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	return result, nil
}

// buildEventConditions converts event patterns to SQL conditions.
// Examples:
//   - "activeflow_created" -> "event_type = 'activeflow_created'"
//   - "activeflow_*" -> "event_type LIKE 'activeflow_%'"
//   - "*" -> "" (no filter)
func buildEventConditions(events []string) string {
	var conditions []string

	for _, e := range events {
		if e == "*" {
			// Wildcard all - no filter needed
			return ""
		}

		if strings.HasSuffix(e, "_*") {
			// Prefix wildcard: "activeflow_*" -> LIKE 'activeflow_%'
			prefix := strings.TrimSuffix(e, "*")
			conditions = append(conditions, fmt.Sprintf("event_type LIKE '%s%%'", prefix))
		} else {
			// Exact match
			conditions = append(conditions, fmt.Sprintf("event_type = '%s'", e))
		}
	}

	return strings.Join(conditions, " OR ")
}
