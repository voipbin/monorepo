package speaking

// Field represents a database field name for Speaking
type Field string

const (
	FieldID            Field = "id"
	FieldCustomerID    Field = "customer_id"
	FieldReferenceType Field = "reference_type"
	FieldReferenceID   Field = "reference_id"
	FieldLanguage      Field = "language"
	FieldProvider      Field = "provider"
	FieldVoiceID       Field = "voice_id"
	FieldDirection     Field = "direction"
	FieldStatus        Field = "status"
	FieldPodID         Field = "pod_id"
	FieldTMCreate      Field = "tm_create"
	FieldTMUpdate      Field = "tm_update"
	FieldTMDelete      Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
