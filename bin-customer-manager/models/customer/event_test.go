package customer

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
)

// CustomerCreatedEvent's subscription address is the embedded Customer's id, via its own
// explicit shadowing method (VOIP-1419) -- never via promotion of *Customer's method, whose
// call would panic on a nil embed.
var _ eventtopic.SubscriptionIdentifier = (*CustomerCreatedEvent)(nil)

func TestCustomerCreatedEventEventSubscriptionID(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())
	billingAccountID := uuid.Must(uuid.NewV4())

	e := &CustomerCreatedEvent{
		Customer: &Customer{
			ID:               customerID,
			BillingAccountID: billingAccountID,
		},
		Headless: true,
	}

	res := e.EventSubscriptionID()
	if res != customerID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", customerID.String(), res)
	}
	if res == billingAccountID.String() {
		t.Errorf("CustomerCreatedEvent must be addressed by the embedded customer's id, not another id field. got: %s", res)
	}
}

// TestCustomerCreatedEventEventSubscriptionIDNilEmbed pins the nil-guard branch: a wrapper
// whose embed is nil must return "" (the `-` placeholder address) WITHOUT panicking.
func TestCustomerCreatedEventEventSubscriptionIDNilEmbed(t *testing.T) {
	e := &CustomerCreatedEvent{
		Headless: true,
	}

	res := e.EventSubscriptionID()
	if res != "" {
		t.Errorf("Wrong match. a nil embed carries no identity. expect: \"\", got: %s", res)
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_customer_created", EventTypeCustomerCreated, "customer_created"},
		{"event_type_customer_updated", EventTypeCustomerUpdated, "customer_updated"},
		{"event_type_customer_deleted", EventTypeCustomerDeleted, "customer_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
