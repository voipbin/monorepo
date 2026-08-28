package direct

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Direct is published on the global topic exchange `bin-manager.event` and must carry an explicit
// subscription address (VOIP-1419). The assertion pins the POINTER type: the event data reaches
// notifyhandler as a pointer and the interface check matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Direct)(nil)

// TestDirectEventSubscriptionID asserts the subscription address is the direct's OWN id — not any
// of the other uuid-typed fields a wrong implementation could plausibly return. ResourceID is the
// plausible wrong answer: a Direct fronts another resource (agent, queue, conference, ...), but a
// consumer following one direct binds the direct's own id, and events about the fronted resource
// come from that resource's own publisher. Every uuid is distinct, so returning the wrong field
// fails loudly (mutation check).
func TestDirectEventSubscriptionID(t *testing.T) {
	directID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	resourceID := uuid.Must(uuid.NewV4())

	data := &Direct{
		Identity: commonidentity.Identity{
			ID:         directID,
			CustomerID: customerID,
		},
		ResourceType: ResourceTypeAgent,
		ResourceID:   resourceID,
		Hash:         "direct.testhash",
	}

	res := data.EventSubscriptionID()
	if res != directID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", directID.String(), res)
	}
	if res == resourceID.String() {
		t.Errorf("Direct must not be addressed by the id of the resource it fronts. got: %s", res)
	}
	if res == customerID.String() {
		t.Errorf("Direct must not be addressed by its customer id. got: %s", res)
	}
}
