package contact

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage addresses its own id on the global topic exchange via the
// EventSubscriptionID promoted from the embedded commonidentity.Identity
// (VOIP-1419). The assertion pins the POINTER type: the event data reaches notifyhandler as a
// POINTER and the assertion matches the dynamic type; a VALUE of this pointer-receiver type
// would fail the assertion.
var _ eventtopic.SubscriptionIdentifier = (*WebhookMessage)(nil)

func Test_WebhookMessage_EventSubscriptionID(t *testing.T) {
	contactID := uuid.FromStringOrNil("a7c01e2d-5001-4001-8001-000000000001")

	tests := []struct {
		name    string
		message *WebhookMessage
		expect  string
	}{
		{
			"normal",
			&WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         contactID,
					CustomerID: uuid.FromStringOrNil("a7c01e2d-5001-4001-8001-000000000002"),
				},
				FirstName: "sungtae",
				LastName:  "kim",
			},
			contactID.String(),
		},
		{
			"empty id",
			&WebhookMessage{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.message.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// Test_WebhookMessage_EventSubscriptionIDIsNotCustomerID pins the id space of the address: the
// contact's OWN id, not the customer id also present in the payload. A customer id is shared by
// every contact of the customer, so addressing by it would fan unrelated contacts into one
// stream.
func Test_WebhookMessage_EventSubscriptionIDIsNotCustomerID(t *testing.T) {
	contactID := uuid.FromStringOrNil("a7c01e2d-5002-4002-8002-000000000001")
	customerID := uuid.FromStringOrNil("a7c01e2d-5002-4002-8002-000000000002")

	m := &WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
	}

	if res := m.EventSubscriptionID(); res != contactID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", contactID, res)
	}
	if m.EventSubscriptionID() == customerID.String() {
		t.Errorf("Subscription address must not be the customer id. customer_id: %s", customerID)
	}
}
