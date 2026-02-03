package contact

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Contact queries.
// Each field corresponds to a filterable database column.
type FieldStruct struct {
	ID          uuid.UUID `filter:"id"`
	CustomerID  uuid.UUID `filter:"customer_id"`
	FirstName   string    `filter:"first_name"`
	LastName    string    `filter:"last_name"`
	DisplayName string    `filter:"display_name"`
	Company     string    `filter:"company"`
	JobTitle    string    `filter:"job_title"`
	Source      string    `filter:"source"`
	ExternalID  string    `filter:"external_id"`
	Deleted     bool      `filter:"deleted"`
}
