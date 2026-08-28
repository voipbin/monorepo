package summary

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Summary is published on the global topic exchange under its own id (VOIP-1419). The
// assertion pins the POINTER type: the event data reaches notifyhandler as a pointer and
// the eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Summary)(nil)

func TestSummaryEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	onEndFlowID := uuid.Must(uuid.NewV4())

	h := &Summary{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ActiveflowID:  activeflowID,
		OnEndFlowID:   onEndFlowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
	}

	res := h.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}

	// The address must be the Summary's OWN id -- any other plausible id field would produce
	// well-formed keys under a wrong address that no binding ever matches.
	if res == customerID.String() {
		t.Errorf("Subscription address must not be the customer id. got: %s", res)
	}
	if res == activeflowID.String() {
		t.Errorf("Subscription address must not be the activeflow id. got: %s", res)
	}
	if res == referenceID.String() {
		t.Errorf("Subscription address must not be the reference id. got: %s", res)
	}
	if res == onEndFlowID.String() {
		t.Errorf("Subscription address must not be the on-end flow id. got: %s", res)
	}
}

func TestSummaryEventSubscriptionIDZeroValue(t *testing.T) {
	h := &Summary{}
	if res := h.EventSubscriptionID(); res != uuid.Nil.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil.String(), res)
	}
}
