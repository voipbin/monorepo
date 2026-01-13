package conferencecall

// Field represents a database field name for conferencecall
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldActiveflowID Field = "activeflow_id" // activeflow_id
	FieldConferenceID Field = "conference_id" // conference_id

	FieldReferenceType Field = "reference_type" // reference_type
	FieldReferenceID   Field = "reference_id"   // reference_id

	FieldStatus Field = "status" // status

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
