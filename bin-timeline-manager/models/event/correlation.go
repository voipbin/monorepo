package event

import (
	"time"

	"github.com/gofrs/uuid"
)

// CorrelatedRow is the raw scan target for the correlation aggregation query.
// ClickHouse driver constraints require scanning LowCardinality(String) columns
// (publisher, data_type) and the materialized resource_id into Go strings;
// conversion to richer types happens at the eventhandler boundary.
type CorrelatedRow struct {
	Publisher  string
	ResourceID string
	DataType   string
	EventTypes []string
	FirstSeen  time.Time
	LastSeen   time.Time
}

// CorrelatedResource is one deduplicated resource derived from events,
// returned in the correlation graph.
type CorrelatedResource struct {
	ID         uuid.UUID `json:"id"`
	DataType   string    `json:"data_type"`
	EventTypes []string  `json:"event_types"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
}

// PublisherGroup groups correlated resources by publisher (service name).
type PublisherGroup struct {
	Publisher string                `json:"publisher"`
	Resources []*CorrelatedResource `json:"resources"`
}
