package outboundconfig

// Field represents a database field name for OutboundConfig
type Field string

const (
	FieldID         Field = "id"          // id
	FieldCustomerID Field = "customer_id" // customer_id

	FieldName                 Field = "name"                  // name
	FieldDetail               Field = "detail"                // detail
	FieldDestinationWhitelist Field = "destination_whitelist" // destination_whitelist
	FieldCodecs               Field = "codecs"                // codecs

	FieldDefaultOutgoingSourceNumberID Field = "default_outgoing_source_number_id" // default_outgoing_source_number_id

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete
)
