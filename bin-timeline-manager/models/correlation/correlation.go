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

// ResourceCorrelation is the domain result of resolving a resource id to its
// activeflow correlation graph. It is the value returned by
// eventHandler.ResourceCorrelationGet; the listenhandler maps it into the
// transport DTO (response.V1DataResourceCorrelationGet) before marshalling.
type ResourceCorrelation struct {
	ResourceID    uuid.UUID
	ResourceFound bool
	ActiveflowID  uuid.UUID
	Truncated     bool
	Resources     []*PublisherGroup
}

// ResourceCorrelationResponse is the transport contract for
// GET /v1/correlations/<id>. It is the single source of truth for the wire
// shape: the listenhandler response DTO (response.V1DataResourceCorrelationGet)
// is a type alias of this type, and the requesthandler client unmarshals into
// it. The domain type above (ResourceCorrelation) carries no json tags and is
// mapped into this type at the listenhandler boundary.
type ResourceCorrelationResponse struct {
	ResourceID    uuid.UUID         `json:"resource_id"`
	ResourceFound bool              `json:"resource_found"`
	ActiveflowID  uuid.UUID         `json:"activeflow_id"`
	Truncated     bool              `json:"truncated"`
	Resources     []*PublisherGroup `json:"resources"`
}
