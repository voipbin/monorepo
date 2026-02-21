package extension

// Field represents a typed field name for extension queries
type Field string

// Field constants for extension
const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldEndpointID Field = "endpoint_id" // endpoint_id
	FieldAORID      Field = "aor_id"      // aor_id
	FieldAuthID     Field = "auth_id"     // auth_id

	FieldExtension  Field = "extension"   // extension
	FieldDomainName Field = "domain_name" // domain_name

	FieldRealm    Field = "realm"    // realm
	FieldUsername Field = "username" // username
	FieldPassword Field = "password" // password

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
