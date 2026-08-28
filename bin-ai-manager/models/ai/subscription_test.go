package ai

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	"monorepo/bin-common-handler/models/identity"
)

// AI is published on the global topic exchange under its own id (VOIP-1419). The assertion
// pins the POINTER type: the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*AI)(nil)

func TestAIEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ragID := uuid.Must(uuid.NewV4())
	directID := uuid.Must(uuid.NewV4())

	h := &AI{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:     "test ai",
		RagID:    ragID,
		DirectID: directID,
	}

	res := h.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}

	// The address must be the AI's OWN id -- any other plausible id field would produce
	// well-formed keys under a wrong address that no binding ever matches.
	if res == customerID.String() {
		t.Errorf("Subscription address must not be the customer id. got: %s", res)
	}
	if res == ragID.String() {
		t.Errorf("Subscription address must not be the rag id. got: %s", res)
	}
	if res == directID.String() {
		t.Errorf("Subscription address must not be the direct id. got: %s", res)
	}
}

func TestAIEventSubscriptionIDZeroValue(t *testing.T) {
	h := &AI{}
	if res := h.EventSubscriptionID(); res != uuid.Nil.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil.String(), res)
	}
}
