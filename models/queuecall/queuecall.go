package queuecall

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

// Queuecall defines
type Queuecall struct {
	ID            uuid.UUID     `json:"id"`
	CustomerID    uuid.UUID     `json:"customer_id"`
	QueueID       uuid.UUID     `json:"queue_id"`
	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	FlowID          uuid.UUID `json:"flow_id"`           // queuecall's queue flow id.
	ForwardActionID uuid.UUID `json:"forward_action_id"` // action id for forward. This is for the confbridge_join's action id.
	ExitActionID    uuid.UUID `json:"exit_action_id"`    // action id for queue exit. When the queuecall has ended, the queuemanager will send the request forward to this action id.
	ConfbridgeID    uuid.UUID `json:"confbridge_id"`     // confbridge id

	Source        cmaddress.Address   `json:"source"`         // source address for calling to the agent.
	RoutingMethod queue.RoutingMethod `json:"routing_method"` // queue's routing method
	TagIDs        []uuid.UUID         `json:"tag_ids"`        // queue's tags

	Status         Status    `json:"status"`
	ServiceAgentID uuid.UUID `json:"service_agent_id"`

	TimeoutWait    int `json:"timeout_wait"`    // timeout for wait.(ms)
	TimeoutService int `json:"timeout_service"` // timeout for service.(service)

	TMCreate  string `json:"tm_create"`  // Created timestamp.
	TMService string `json:"tm_service"` // Serviced timestamp.
	TMUpdate  string `json:"tm_update"`  // Updated timestamp.
	TMDelete  string `json:"tm_delete"`  // Deleted timestamp.
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
	StatusWait      Status = "wait"      // queue call is waiting in the wait actions.
	StatusEntering  Status = "entering"  // queue call is entering to the queue's confbridge
	StatusService   Status = "service"   // queue call is being service now.
	StatusDone      Status = "done"      // queue call done.
	StatusAbandoned Status = "abandoned" // queue call has been abandoned.
)
