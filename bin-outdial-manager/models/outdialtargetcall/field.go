package outdialtargetcall

// Field type for typed field constants
type Field string

// Field constants for outdialtargetcall
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldCampaignID      Field = "campaign_id"
	FieldOutdialID       Field = "outdial_id"
	FieldOutdialTargetID Field = "outdial_target_id"

	FieldActiveflowID  Field = "activeflow_id"
	FieldReferenceType Field = "reference_type"
	FieldReferenceID   Field = "reference_id"

	FieldStatus Field = "status"

	FieldDestination      Field = "destination"
	FieldDestinationIndex Field = "destination_index"
	FieldTryCount         Field = "try_count"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
