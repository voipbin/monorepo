package queue

import (
	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Queue defines
type Queue struct {
	ID         uuid.UUID `json:"id"`          // queue id
	CustomerID uuid.UUID `json:"customer_id"` // owner id

	// basic info
	Name   string `json:"name"`   // queue's name
	Detail string `json:"detail"` // queue's detail

	// operation info
	RoutingMethod RoutingMethod `json:"routing_method"` // queue's routing method
	TagIDs        []uuid.UUID   `json:"tag_ids"`        // queue's tag ids

	// execute
	Execute Execute `json:"execute"`

	// wait/service info
	WaitActions    []fmaction.Action `json:"wait_actions"`    // actions for queue waiting
	WaitTimeout    int               `json:"wait_timeout"`    // wait queue timeout.(ms)
	ServiceTimeout int               `json:"service_timeout"` // service queue timeout(ms).

	// queuecall info
	WaitQueuecallIDs    []uuid.UUID `json:"wait_queuecall_ids"`    // waiting queue call ids.
	ServiceQueuecallIDs []uuid.UUID `json:"service_queuecall_ids"` // service queue call ids(ms).

	TotalIncomingCount  int `json:"total_incoming_count"`  // total incoming call count
	TotalServicedCount  int `json:"total_serviced_count"`  // total serviced call count
	TotalAbandonedCount int `json:"total_abandoned_count"` // total abandoned call count

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
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
