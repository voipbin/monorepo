package queuecall

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Queuecall queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID             uuid.UUID     `filter:"id"`
	CustomerID     uuid.UUID     `filter:"customer_id"`
	QueueID        uuid.UUID     `filter:"queue_id"`
	ReferenceType  ReferenceType `filter:"reference_type"`
	ReferenceID    uuid.UUID     `filter:"reference_id"`
	Source         string        `filter:"source"`
	Status         Status        `filter:"status"`
	ServiceAgentID uuid.UUID     `filter:"service_agent_id"`
	Deleted        bool          `filter:"deleted"`
}
