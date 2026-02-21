package provider

// Field represents database field names for Provider
type Field string

const (
	FieldID Field = "id" // id

	FieldType     Field = "type"     // type
	FieldHostname Field = "hostname" // hostname

	FieldTechPrefix  Field = "tech_prefix"  // tech_prefix
	FieldTechPostfix Field = "tech_postfix" // tech_postfix
	FieldTechHeaders Field = "tech_headers" // tech_headers

	FieldName   Field = "name"   // name
	FieldDetail Field = "detail" // detail

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
