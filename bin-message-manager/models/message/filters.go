package message

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Message queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID                   uuid.UUID    `filter:"id"`
	CustomerID           uuid.UUID    `filter:"customer_id"`
	Type                 Type         `filter:"type"`
	Source               string       `filter:"source"`
	ProviderName         ProviderName `filter:"provider_name"`
	ProviderReferenceID  string       `filter:"provider_reference_id"`
	Direction            Direction    `filter:"direction"`
	Deleted              bool         `filter:"deleted"`
}
