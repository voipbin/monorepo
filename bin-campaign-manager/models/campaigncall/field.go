package campaigncall

// Field represents a database field name for Campaigncall
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldCampaignID Field = "campaign_id" // campaign_id

	FieldOutplanID       Field = "outplan_id"        // outplan_id
	FieldOutdialID       Field = "outdial_id"        // outdial_id
	FieldOutdialTargetID Field = "outdial_target_id" // outdial_target_id
	FieldQueueID         Field = "queue_id"          // queue_id

	FieldActiveflowID Field = "activeflow_id" // activeflow_id
	FieldFlowID       Field = "flow_id"       // flow_id

	FieldReferenceType Field = "reference_type" // reference_type
	FieldReferenceID   Field = "reference_id"   // reference_id

	FieldStatus Field = "status" // status
	FieldResult Field = "result" // result

	FieldSource           Field = "source"            // source
	FieldDestination      Field = "destination"       // destination
	FieldDestinationIndex Field = "destination_index" // destination_index
	FieldTryCount         Field = "try_count"         // try_count

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
