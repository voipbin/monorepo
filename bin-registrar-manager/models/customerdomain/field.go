package customerdomain

// Field represents a typed field name for customer domain queries
type Field string

// Field constants for customer domain
const (
	FieldCustomerID Field = "customer_id" // customer_id

	FieldDomainLabel Field = "domain_label" // domain_label
	FieldRealm       Field = "realm"        // realm

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
)
