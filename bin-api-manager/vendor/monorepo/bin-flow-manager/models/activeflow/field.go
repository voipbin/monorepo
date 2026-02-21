package activeflow

type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldFlowID Field = "flow_id" // flow_id
	FieldStatus Field = "status"  // status

	FieldReferenceType         Field = "reference_type"          // reference_type
	FieldReferenceID           Field = "reference_id"            // reference_id
	FieldReferenceActiveflowID Field = "reference_activeflow_id" // reference_activeflow_id

	FieldOnCompleteFlowID Field = "on_complete_flow_id" // on_complete_flow_id

	FieldStackMap       Field = "stack_map"        // stack_map
	FieldCurrentStackID Field = "current_stack_id" // current_stack_id
	FieldCurrentAction  Field = "current_action"   // current_action

	FieldForwardStackID  Field = "forward_stack_id"  // forward_stack_id
	FieldForwardActionID Field = "forward_action_id" // forward_action_id

	FieldExecuteCount    Field = "execute_count"    // execute_count
	FieldExecutedActions Field = "executed_actions" // executed_actions

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
