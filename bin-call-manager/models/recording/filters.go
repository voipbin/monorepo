package recording

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Recording queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID    uuid.UUID     `filter:"customer_id"`
	OwnerID       uuid.UUID     `filter:"owner_id"`
	ActiveflowID  uuid.UUID     `filter:"activeflow_id"`
	ReferenceType ReferenceType `filter:"reference_type"`
	ReferenceID   uuid.UUID     `filter:"reference_id"`
	Status        Status        `filter:"status"`
	Format        Format        `filter:"format"`
	OnEndFlowID   uuid.UUID     `filter:"on_end_flow_id"`
	AsteriskID    string        `filter:"asterisk_id"`
	Deleted       bool          `filter:"deleted"`
}
