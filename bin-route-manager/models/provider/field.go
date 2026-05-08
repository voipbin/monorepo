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

	FieldMetadata Field = "metadata" // metadata

	FieldCodecs Field = "codecs" // codecs

	FieldHealthStatus    Field = "health_status"    // health_status
	FieldHealthCheckedAt Field = "health_checked_at" // health_checked_at

	FieldTMCreate Field = "tm_create" // tm_create
	FieldTMUpdate Field = "tm_update" // tm_update
	FieldTMDelete Field = "tm_delete" // tm_delete

	// filter only
	FieldDeleted Field = "deleted"
)
