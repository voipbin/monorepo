package outdial

// Field type for typed field constants
type Field string

// Field constants for outdial
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldCampaignID Field = "campaign_id"

	FieldName   Field = "name"
	FieldDetail Field = "detail"

	FieldData Field = "data"

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)
