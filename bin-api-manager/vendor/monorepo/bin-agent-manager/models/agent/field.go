package agent

// Field type for typed field maps
type Field string

// list of fields
const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldUsername     Field = "username"      // username
	FieldPasswordHash Field = "password_hash" // password_hash

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldRingMethod Field = "ring_method" // ring_method

	FieldStatus     Field = "status"     // status
	FieldPermission Field = "permission" // permission
	FieldTagIDs     Field = "tag_ids"    // tag_ids
	FieldAddresses  Field = "addresses"  // addresses

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted" // deleted
)
