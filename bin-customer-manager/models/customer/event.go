package customer

// list of event types
const (
	EventTypeCustomerCreated string = "customer_created" // the customer has created
	EventTypeCustomerUpdated string = "customer_updated" // the customer's info has updated
	EventTypeCustomerDeleted string = "customer_deleted" // the customer has deleted
)

// CustomerCreatedEvent wraps the Customer with headless flag for the customer_created event.
type CustomerCreatedEvent struct {
	*Customer
	Headless bool `json:"headless"`
}
