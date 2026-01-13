package outdialtarget

// Field type for typed field constants
type Field string

// Field constants for outdialtarget
const (
	FieldID        Field = "id"
	FieldOutdialID Field = "outdial_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldData   Field = "data"
	FieldStatus Field = "status"

	FieldDestination0 Field = "destination_0"
	FieldDestination1 Field = "destination_1"
	FieldDestination2 Field = "destination_2"
	FieldDestination3 Field = "destination_3"
	FieldDestination4 Field = "destination_4"

	FieldTryCount0 Field = "try_count_0"
	FieldTryCount1 Field = "try_count_1"
	FieldTryCount2 Field = "try_count_2"
	FieldTryCount3 Field = "try_count_3"
	FieldTryCount4 Field = "try_count_4"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
