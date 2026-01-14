package queue

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Queue defines
type Queue struct {
	commonidentity.Identity

	// basic info
	Name   string `json:"name,omitempty" db:"name"`     // queue's name
	Detail string `json:"detail,omitempty" db:"detail"` // queue's detail

	// operation info
	RoutingMethod RoutingMethod `json:"routing_method,omitempty" db:"routing_method"` // queue's routing method
	TagIDs        []uuid.UUID   `json:"tag_ids,omitempty" db:"tag_ids,json"`          // queue's tag ids

	// execute
	Execute Execute `json:"execute,omitempty" db:"execute"`

	// wait/service info
	WaitFlowID     uuid.UUID `json:"wait_flow_id,omitempty" db:"wait_flow_id,uuid"`    // flow id for queue waiting
	WaitTimeout    int       `json:"wait_timeout,omitempty" db:"wait_timeout"`         // wait queue timeout.(ms)
	ServiceTimeout int       `json:"service_timeout,omitempty" db:"service_timeout"`   // service queue timeout(ms).

	// queuecall info
	WaitQueuecallIDs    []uuid.UUID `json:"wait_queuecall_ids,omitempty" db:"wait_queue_call_ids,json"`       // waiting queue call ids.
	ServiceQueuecallIDs []uuid.UUID `json:"service_queuecall_ids,omitempty" db:"service_queue_call_ids,json"` // service queue call ids(ms).

	TotalIncomingCount  int `json:"total_incoming_count,omitempty" db:"total_incoming_count"`   // total incoming call count
	TotalServicedCount  int `json:"total_serviced_count,omitempty" db:"total_serviced_count"`   // total serviced call count
	TotalAbandonedCount int `json:"total_abandoned_count,omitempty" db:"total_abandoned_count"` // total abandoned call count

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"` // Deleted timestamp.
}

// RoutingMethod type
type RoutingMethod string

// list of routing methods
const (
	RoutingMethodNone   RoutingMethod = ""
	RoutingMethodRandom RoutingMethod = "random"
)

// Execute defines
type Execute string

// list of executes
const (
	ExecuteRun  Execute = "run"
	ExecuteStop Execute = "stop"
)
