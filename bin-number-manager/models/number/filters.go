package number

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Number queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID                   uuid.UUID    `filter:"id"`
	CustomerID           uuid.UUID    `filter:"customer_id"`
	Number               string       `filter:"number"`
	CallFlowID           uuid.UUID    `filter:"call_flow_id"`
	MessageFlowID        uuid.UUID    `filter:"message_flow_id"`
	Name                 string       `filter:"name"`
	ProviderName         ProviderName `filter:"provider_name"`
	ProviderReferenceID  string       `filter:"provider_reference_id"`
	Status               Status       `filter:"status"`
	T38Enabled           bool         `filter:"t38_enabled"`
	EmergencyEnabled     bool         `filter:"emergency_enabled"`
	Deleted              bool         `filter:"deleted"`
}
