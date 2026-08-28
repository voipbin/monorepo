package providercall

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
)

// ProviderCall publishes on the global topic exchange (VOIP-1404/1419), so it must carry an
// explicit subscription address. The assertion pins the POINTER type: the event data reaches
// notifyhandler as a pointer and the interface assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*ProviderCall)(nil)

func TestProviderCallEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	providerID := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())

	p := &ProviderCall{
		ID:         id,
		CustomerID: customerID,
		ProviderID: providerID,
		FlowID:     flowID,
	}

	res := p.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}
	// A providercall is an independent top-level resource: its address must be its own id,
	// never one of the resources it references.
	if res == customerID.String() {
		t.Errorf("ProviderCall must not be addressed by its customer id. got: %s", res)
	}
	if res == providerID.String() {
		t.Errorf("ProviderCall must not be addressed by its provider id. got: %s", res)
	}
	if res == flowID.String() {
		t.Errorf("ProviderCall must not be addressed by its flow id. got: %s", res)
	}
}
