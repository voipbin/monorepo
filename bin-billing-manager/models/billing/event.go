package billing

// list of event types
const (
	EventTypeBillingCreated string = "billing_created" // the billing has created
	EventTypeBillingUpdated string = "billing_updated" // the billing's info has updated
	EventTypeBillingDeleted string = "billing_deleted" // the billing's info has deleted
)
