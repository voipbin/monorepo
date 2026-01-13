package queuecall

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-queue-manager/models/queue"
)

// Queuecall defines
type Queuecall struct {
	commonidentity.Identity

	QueueID uuid.UUID `json:"queue_id,omitempty" db:"queue_id,uuid"`

	ReferenceType         ReferenceType `json:"reference_type,omitempty" db:"reference_type"`                    // referenced resource's type.
	ReferenceID           uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`                   // referenced resource's id.
	ReferenceActiveflowID uuid.UUID     `json:"reference_activeflow_id,omitempty" db:"reference_activeflow_id,uuid"` // referenced resource's activeflow id

	ForwardActionID uuid.UUID `json:"forward_action_id,omitempty" db:"forward_action_id,uuid"` // action id for forward. This is for the conference_join's action id.
	ConfbridgeID    uuid.UUID `json:"confbridge_id,omitempty" db:"confbridge_id,uuid"`         // confbridge id

	Source        commonaddress.Address `json:"source,omitempty" db:"source,json"`           // source address for calling to the agent.
	RoutingMethod queue.RoutingMethod   `json:"routing_method,omitempty" db:"routing_method"` // queue's routing method
	TagIDs        []uuid.UUID           `json:"tag_ids,omitempty" db:"tag_ids,json"`          // queue's tags

	Status         Status    `json:"status,omitempty" db:"status"`
	ServiceAgentID uuid.UUID `json:"service_agent_id,omitempty" db:"service_agent_id,uuid"`

	TimeoutWait    int `json:"timeout_wait,omitempty" db:"timeout_wait"`       // timeout for wait.(ms)
	TimeoutService int `json:"timeout_service,omitempty" db:"timeout_service"` // timeout for service.(ms)

	DurationWaiting int `json:"duration_waiting,omitempty" db:"duration_waiting"` // duration for waiting(ms)
	DurationService int `json:"duration_service,omitempty" db:"duration_service"` // duration for service(ms)

	TMCreate  string `json:"tm_create,omitempty" db:"tm_create"`   // Created timestamp.
	TMService string `json:"tm_service,omitempty" db:"tm_service"` // Serviced timestamp.
	TMUpdate  string `json:"tm_update,omitempty" db:"tm_update"`   // Updated timestamp.
	TMEnd     string `json:"tm_end,omitempty" db:"tm_end"`         // ended timestamp.
	TMDelete  string `json:"tm_delete,omitempty" db:"tm_delete"`   // Deleted timestamp.
}

// ReferenceType define
type ReferenceType string

// list of reference types.
const (
	ReferenceTypeCall = "call"
)

// Status define
type Status string

// list of status
const (
	StatusInitiating Status = "initiating" // queue call is initiating.
	StatusWaiting    Status = "waiting"    // queue call is waiting in the wait actions.
	StatusConnecting Status = "connecting" // queue call is connecting to the agent.
	StatusKicking    Status = "kicking"    // queue call is being kick from the queue
	StatusService    Status = "service"    // queue call is being service now.
	StatusDone       Status = "done"       // queue call done.
	StatusAbandoned  Status = "abandoned"  // queue call has been abandoned.
)
