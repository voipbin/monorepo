package correlation

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

// Correlation is the result of resolving a resource id to its activeflow and
// the correlation graph of all resources sharing that activeflow, grouped by
// publisher. It carries json tags and is serialized directly on the wire by the
// listenhandler and unmarshaled directly by the requesthandler client (the
// flow-manager Flow pattern: the model is the transport shape, no separate DTO).
type Correlation struct {
	ResourceID    uuid.UUID         `json:"resource_id"`
	ResourceFound bool              `json:"resource_found"`
	ActiveflowID  uuid.UUID         `json:"activeflow_id"`
	Truncated     bool              `json:"truncated"`
	Resources     []*PublisherGroup `json:"resources"`
}
