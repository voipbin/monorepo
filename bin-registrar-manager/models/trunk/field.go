package trunk

// Field represents a typed field name for trunk queries
type Field string

// Field constants for trunk
const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldDomainName Field = "domain_name" // domain_name

	FieldAuthTypes  Field = "auth_types"  // auth_types
	FieldRealm      Field = "realm"       // realm
	FieldUsername   Field = "username"    // username
	FieldPassword   Field = "password"    // password
	FieldAllowedIPs Field = "allowed_ips" // allowed_ips

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
