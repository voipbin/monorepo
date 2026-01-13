package outplan

// Field represents a database field name for Outplan
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldSource Field = "source" // source

	FieldDialTimeout Field = "dial_timeout" // dial_timeout
	FieldTryInterval Field = "try_interval" // try_interval

	FieldMaxTryCount0 Field = "max_try_count_0" // max_try_count_0
	FieldMaxTryCount1 Field = "max_try_count_1" // max_try_count_1
	FieldMaxTryCount2 Field = "max_try_count_2" // max_try_count_2
	FieldMaxTryCount3 Field = "max_try_count_3" // max_try_count_3
	FieldMaxTryCount4 Field = "max_try_count_4" // max_try_count_4

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
