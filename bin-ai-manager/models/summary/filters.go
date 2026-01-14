package summary

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Summary queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID    uuid.UUID     `filter:"customer_id"`
	ActiveflowID  uuid.UUID     `filter:"activeflow_id"`
	OnEndFlowID   uuid.UUID     `filter:"on_end_flow_id"`
	ReferenceType ReferenceType `filter:"reference_type"`
	ReferenceID   uuid.UUID     `filter:"reference_id"`
	Status        Status        `filter:"status"`
	Language      string        `filter:"language"`
	Deleted       bool          `filter:"deleted"`
}
