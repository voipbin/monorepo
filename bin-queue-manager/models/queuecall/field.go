package queuecall

// Field represents database field names for Queuecall
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldQueueID Field = "queue_id" // queue_id

	FieldReferenceType         Field = "reference_type"          // reference_type
	FieldReferenceID           Field = "reference_id"            // reference_id
	FieldReferenceActiveflowID Field = "reference_activeflow_id" // reference_activeflow_id

	FieldForwardActionID Field = "forward_action_id" // forward_action_id
	FieldConfbridgeID    Field = "confbridge_id"     // confbridge_id

	FieldSource        Field = "source"         // source
	FieldRoutingMethod Field = "routing_method" // routing_method
	FieldTagIDs        Field = "tag_ids"        // tag_ids

	FieldStatus         Field = "status"           // status
	FieldServiceAgentID Field = "service_agent_id" // service_agent_id

	FieldTimeoutWait    Field = "timeout_wait"    // timeout_wait
	FieldTimeoutService Field = "timeout_service" // timeout_service

	FieldDurationWaiting Field = "duration_waiting" // duration_waiting
	FieldDurationService Field = "duration_service" // duration_service

	FieldTMCreate  Field = "tm_create"  // tm_create
	FieldTMService Field = "tm_service" // tm_service
	FieldTMUpdate  Field = "tm_update"  // tm_update
	FieldTMEnd     Field = "tm_end"     // tm_end
	FieldTMDelete  Field = "tm_delete"  // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
