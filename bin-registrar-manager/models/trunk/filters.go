package trunk

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Trunk queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID          uuid.UUID `filter:"id"`
	CustomerID  uuid.UUID `filter:"customer_id"`
	Name        string    `filter:"name"`
	DomainName  string    `filter:"domain_name"`
	Realm       string    `filter:"realm"`
	Username    string    `filter:"username"`
	Deleted     bool      `filter:"deleted"`
}
