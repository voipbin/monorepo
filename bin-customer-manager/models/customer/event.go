package customer

// list of event types
const (
	EventTypeCustomerCreated string = "customer_created" // the customer has created
	EventTypeCustomerUpdated string = "customer_updated" // the customer's info has updated
	EventTypeCustomerDeleted   string = "customer_deleted"   // the customer has deleted
	EventTypeCustomerFrozen    string = "customer_frozen"    // the customer has been frozen
	EventTypeCustomerRecovered                   string = "customer_recovered"                    // the customer has been recovered
	EventTypeCustomerIdentityVerificationUpdated string = "customer_identity_verification_updated" // the customer's identity verification has updated
)

// CustomerCreatedEvent wraps the Customer with headless flag for the customer_created event.
type CustomerCreatedEvent struct {
	*Customer
	Headless bool `json:"headless"`
}

// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event`: the embedded Customer's id (VOIP-1404 §4.2, VOIP-1419).
// The explicit shadowing method with its nil guard is mandatory: the anonymous *Customer
// embed would otherwise promote a method whose call panics on a nil embed.
func (h *CustomerCreatedEvent) EventSubscriptionID() string {
	if h.Customer == nil {
		return ""
	}
	//nolint:staticcheck // QF1008: the explicit embedded selector is deliberate -- it documents that the address is the embedded Customer's id, guarded right above.
	return h.Customer.ID.String()
}
