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

	QueueID uuid.UUID `json:"queue_id,omitempty"`

	ReferenceType         ReferenceType `json:"reference_type,omitempty"`          // referenced resource's type.
	ReferenceID           uuid.UUID     `json:"reference_id,omitempty"`            // referenced resource's id.
	ReferenceActiveflowID uuid.UUID     `json:"reference_activeflow_id,omitempty"` // referenced resource's activeflow id

	ForwardActionID uuid.UUID `json:"forward_action_id,omitempty"` // action id for forward. This is for the conference_join's action id.
	ConfbridgeID    uuid.UUID `json:"confbridge_id,omitempty"`     // confbridge id

	Source        commonaddress.Address `json:"source,omitempty"`         // source address for calling to the agent.
	RoutingMethod queue.RoutingMethod   `json:"routing_method,omitempty"` // queue's routing method
	TagIDs        []uuid.UUID           `json:"tag_ids,omitempty"`        // queue's tags

	Status         Status    `json:"status,omitempty"`
	ServiceAgentID uuid.UUID `json:"service_agent_id,omitempty"`

	TimeoutWait    int `json:"timeout_wait,omitempty"`    // timeout for wait.(ms)
	TimeoutService int `json:"timeout_service,omitempty"` // timeout for service.(ms)

	DurationWaiting int `json:"duration_waiting,omitempty"` // duration for waiting(ms)
	DurationService int `json:"duration_service,omitempty"` // duration for service(ms)

	TMCreate  string `json:"tm_create,omitempty"`  // Created timestamp.
	TMService string `json:"tm_service,omitempty"` // Serviced timestamp.
	TMUpdate  string `json:"tm_update,omitempty"`  // Updated timestamp.
	TMEnd     string `json:"tm_end,omitempty"`     // ended timestamp.
	TMDelete  string `json:"tm_delete,omitempty"`  // Deleted timestamp.
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
