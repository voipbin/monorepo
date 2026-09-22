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

	FieldStatus     Field = "status"      // status
	FieldPermission Field = "permission"  // permission
	FieldTagIDs     Field = "tag_ids"     // tag_ids
	FieldAddresses  Field = "addresses"   // addresses
	FieldDirectID   Field = "direct_id"   // direct_id
	FieldDirectHash Field = "direct_hash" // direct_hash

	FieldReserveReferenceType Field = "reserve_reference_type" // reserve_reference_type
	FieldReserveReferenceID   Field = "reserve_reference_id"   // reserve_reference_id
	FieldTMReserve            Field = "tm_reserve"             // tm_reserve

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted" // deleted
)
