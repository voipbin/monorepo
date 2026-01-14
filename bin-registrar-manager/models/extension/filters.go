package extension

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Extension queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID          uuid.UUID `filter:"id"`
	CustomerID  uuid.UUID `filter:"customer_id"`
	Name        string    `filter:"name"`
	EndpointID  string    `filter:"endpoint_id"`
	AORID       string    `filter:"aor_id"`
	AuthID      string    `filter:"auth_id"`
	Extension   string    `filter:"extension"`
	DomainName  string    `filter:"domain_name"`
	Realm       string    `filter:"realm"`
	Username    string    `filter:"username"`
	Deleted     bool      `filter:"deleted"`
}
