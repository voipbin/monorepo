package queue

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Queue defines
type Queue struct {
	commonidentity.Identity

	// basic info
	Name   string `json:"name,omitempty"`   // queue's name
	Detail string `json:"detail,omitempty"` // queue's detail

	// operation info
	RoutingMethod RoutingMethod `json:"routing_method,omitempty"` // queue's routing method
	TagIDs        []uuid.UUID   `json:"tag_ids,omitempty"`        // queue's tag ids

	// execute
	Execute Execute `json:"execute,omitempty"`

	// wait/service info
	WaitFlowID     uuid.UUID `json:"wait_flow_id,omitempty"`    // flow id for queue waiting
	WaitTimeout    int       `json:"wait_timeout,omitempty"`    // wait queue timeout.(ms)
	ServiceTimeout int       `json:"service_timeout,omitempty"` // service queue timeout(ms).

	// queuecall info
	WaitQueuecallIDs    []uuid.UUID `json:"wait_queuecall_ids,omitempty"`    // waiting queue call ids.
	ServiceQueuecallIDs []uuid.UUID `json:"service_queuecall_ids,omitempty"` // service queue call ids(ms).

	TotalIncomingCount  int `json:"total_incoming_count,omitempty"`  // total incoming call count
	TotalServicedCount  int `json:"total_serviced_count,omitempty"`  // total serviced call count
	TotalAbandonedCount int `json:"total_abandoned_count,omitempty"` // total abandoned call count

	TMCreate string `json:"tm_create,omitempty"` // Created timestamp.
	TMUpdate string `json:"tm_update,omitempty"` // Updated timestamp.
	TMDelete string `json:"tm_delete,omitempty"` // Deleted timestamp.
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
