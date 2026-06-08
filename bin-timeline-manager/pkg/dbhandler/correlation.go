package dbhandler

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/event"
)

// max_execution_time (seconds) applied to the correlation lookups via the
// per-query context. These are point lookups backed by skip indexes; the
// timeout guards against a pathological full scan. ClickHouse does not accept
// positional bind parameters in the SETTINGS clause, so the limit is passed
// through clickhouse.WithSettings on the query context instead.
const (
	correlationLookupTimeout = 5
	correlationListTimeout   = 10
)

// withQueryTimeout returns a context carrying a ClickHouse max_execution_time
// setting (in seconds) for a single query.
func withQueryTimeout(ctx context.Context, seconds int) context.Context {
	return clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{
		"max_execution_time": seconds,
	}))
}

// ResourceActiveflowIDGet returns the activeflow_id for a given resource_id.
// Returns "" (no error) when no matching event with a non-empty activeflow_id exists.
// ORDER BY timestamp ASC makes the result deterministic even if a resource_id were
// (abnormally) associated with more than one activeflow.
func (h *dbHandler) ResourceActiveflowIDGet(ctx context.Context, resourceID string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ResourceActiveflowIDGet",
		"resource_id": resourceID,
	})

	if h.conn == nil {
		return "", errors.New("clickhouse connection not established")
	}

	query := `
		SELECT activeflow_id
		FROM events
		WHERE resource_id = ?
		  AND activeflow_id != ''
		ORDER BY timestamp ASC
		LIMIT 1
	`

	rows, err := h.conn.Query(withQueryTimeout(ctx, correlationLookupTimeout), query, resourceID)
	if err != nil {
		return "", errors.Wrap(err, "could not query resource activeflow_id")
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", errors.Wrap(err, "error iterating rows")
		}
		log.Debugf("No activeflow_id found for resource.")
		return "", nil
	}

	var activeflowID string
	if err := rows.Scan(&activeflowID); err != nil {
		return "", errors.Wrap(err, "could not scan activeflow_id")
	}

	return activeflowID, nil
}

// ResourceExists reports whether any event row exists for the resource_id.
// Used to distinguish "resource never seen" from "resource has no activeflow".
func (h *dbHandler) ResourceExists(ctx context.Context, resourceID string) (bool, error) {
	if h.conn == nil {
		return false, errors.New("clickhouse connection not established")
	}

	query := `
		SELECT 1
		FROM events
		WHERE resource_id = ?
		LIMIT 1
	`

	rows, err := h.conn.Query(withQueryTimeout(ctx, correlationLookupTimeout), query, resourceID)
	if err != nil {
		return false, errors.Wrap(err, "could not query resource existence")
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, errors.Wrap(err, "error iterating rows")
		}
		return false, nil
	}

	return true, nil
}

// CorrelatedResourceList returns deduplicated resources grouped by (publisher,
// resource_id) for a given activeflow_id, aggregated at the ClickHouse layer.
// limit caps the row count (callers pass maxResources+1 to detect truncation).
func (h *dbHandler) CorrelatedResourceList(ctx context.Context, activeflowID string, limit int) ([]*event.CorrelatedRow, error) {
	if h.conn == nil {
		return nil, errors.New("clickhouse connection not established")
	}

	query := `
		SELECT
			publisher,
			resource_id,
			min(data_type)             AS data_type,
			groupUniqArray(event_type) AS event_types,
			min(timestamp)             AS first_seen,
			max(timestamp)             AS last_seen
		FROM events
		WHERE activeflow_id = ?
		  AND resource_id != ''
		GROUP BY publisher, resource_id
		ORDER BY first_seen ASC
		LIMIT ?
	`

	rows, err := h.conn.Query(withQueryTimeout(ctx, correlationListTimeout), query, activeflowID, limit)
	if err != nil {
		return nil, errors.Wrap(err, "could not query correlated resources")
	}
	defer func() { _ = rows.Close() }()

	result := []*event.CorrelatedRow{}
	for rows.Next() {
		var r event.CorrelatedRow
		if err := rows.Scan(&r.Publisher, &r.ResourceID, &r.DataType, &r.EventTypes, &r.FirstSeen, &r.LastSeen); err != nil {
			return nil, errors.Wrap(err, "could not scan correlated row")
		}
		result = append(result, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "error iterating rows")
	}

	return result, nil
}
