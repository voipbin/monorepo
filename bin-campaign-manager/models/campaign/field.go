package campaign

// Field represents a database field name for Campaign
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldType    Field = "type"    // type
	FieldExecute Field = "execute" // execute

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldStatus       Field = "status"        // status
	FieldServiceLevel Field = "service_level" // service_level
	FieldEndHandle    Field = "end_handle"    // end_handle

	FieldFlowID  Field = "flow_id" // flow_id
	FieldActions Field = "actions" // actions

	FieldOutplanID      Field = "outplan_id"       // outplan_id
	FieldOutdialID      Field = "outdial_id"       // outdial_id
	FieldQueueID        Field = "queue_id"         // queue_id
	FieldNextCampaignID Field = "next_campaign_id" // next_campaign_id

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
