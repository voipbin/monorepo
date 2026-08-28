package aicall

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	"monorepo/bin-common-handler/models/identity"
)

// AIcall is published on the global topic exchange under its own id (VOIP-1419). The
// assertion pins the POINTER type: the event data reaches notifyhandler as a pointer and
// the eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*AIcall)(nil)

func TestAIcallEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	confbridgeID := uuid.Must(uuid.NewV4())

	h := &AIcall{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ActiveflowID:  activeflowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		ConfbridgeID:  confbridgeID,
		Status:        StatusProgressing,
	}

	res := h.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}

	// The address must be the AIcall's OWN id -- any other plausible id field would produce
	// well-formed keys under a wrong address that no binding ever matches.
	if res == customerID.String() {
		t.Errorf("Subscription address must not be the customer id. got: %s", res)
	}
	if res == referenceID.String() {
		t.Errorf("Subscription address must not be the reference id. got: %s", res)
	}
	if res == activeflowID.String() {
		t.Errorf("Subscription address must not be the activeflow id. got: %s", res)
	}
	if res == confbridgeID.String() {
		t.Errorf("Subscription address must not be the confbridge id. got: %s", res)
	}
}

func TestAIcallEventSubscriptionIDZeroValue(t *testing.T) {
	h := &AIcall{}
	if res := h.EventSubscriptionID(); res != uuid.Nil.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil.String(), res)
	}
}
