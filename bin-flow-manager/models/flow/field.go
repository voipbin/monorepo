package flow

type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldType Field = "type" // type

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldPersist Field = "persist" // persist

	FieldActions Field = "actions" // actions

	FieldOnCompleteFlowID Field = "on_complete_flow_id" // on_complete_flow_id

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
