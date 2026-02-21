package queue

// Field represents database field names for Queue
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldRoutingMethod Field = "routing_method" // routing_method
	FieldTagIDs        Field = "tag_ids"        // tag_ids

	FieldExecute Field = "execute" // execute

	FieldWaitFlowID     Field = "wait_flow_id"     // wait_flow_id
	FieldWaitTimeout    Field = "wait_timeout"     // wait_timeout
	FieldServiceTimeout Field = "service_timeout"  // service_timeout

	FieldWaitQueuecallIDs    Field = "wait_queue_call_ids"    // wait_queue_call_ids
	FieldServiceQueuecallIDs Field = "service_queue_call_ids" // service_queue_call_ids

	FieldTotalIncomingCount  Field = "total_incoming_count"  // total_incoming_count
	FieldTotalServicedCount  Field = "total_serviced_count"  // total_serviced_count
	FieldTotalAbandonedCount Field = "total_abandoned_count" // total_abandoned_count

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
