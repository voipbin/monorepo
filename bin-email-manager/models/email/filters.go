package email

import "github.com/gofrs/uuid"

// FieldStruct defines the allowed filters for Email list queries.
// Each field corresponds to a filterable database column. The `filter:` tag is
// both the wire key sent by callers (api-manager) and the DB column name applied
// by databasehandler.ApplyFields.
//
// JSON columns (source, destinations, attachments) and large free-text columns
// (subject, content) are intentionally excluded — they are not equality-filter
// targets.
//
// The `filter:` tags must stay in sync with the corresponding Field constants in
// field.go (e.g. FieldCustomerID = "customer_id").
type FieldStruct struct {
	ID                  uuid.UUID    `filter:"id"`
	CustomerID          uuid.UUID    `filter:"customer_id"`
	ActiveflowID        uuid.UUID    `filter:"activeflow_id"`
	ProviderType        ProviderType `filter:"provider_type"`
	ProviderReferenceID string       `filter:"provider_reference_id"`
	Status              Status       `filter:"status"`
	Deleted             bool         `filter:"deleted"`
}
