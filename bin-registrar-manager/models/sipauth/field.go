package sipauth

// Field represents a typed field name for sipauth queries
type Field string

// Field constants for sipauth
const (
	FieldID            Field = "id"             // id
	FieldReferenceType Field = "reference_type" // reference_type

	FieldAuthTypes Field = "auth_types" // auth_types
	FieldRealm     Field = "realm"      // realm

	FieldUsername Field = "username" // username
	FieldPassword Field = "password" // password

	FieldAllowedIPs Field = "allowed_ips" // allowed_ips

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
)
