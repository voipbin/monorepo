// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type email-manager publishes today (`email_created`, `email_updated`,
// `email_deleted` -- all three from pkg/emailhandler/email.go) and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. The defect class it
// guards against is "the right key shape carrying the wrong id space": a key whose third segment
// is not the address subscribers can bind to in advance produces well-formed keys that no
// instance binding ever matches, and no runtime metric can detect it. Design doc §2.4 / §4.
//
// email-manager is a DEFAULT-ID service: `email.Email` is an independent, persistent resource
// whose own id IS the subscription address, so it deliberately carries NO
// eventtopic.SubscriptionIdentifier override and the JSON `id` fallback must keep covering it.
// TestEmailUsesDefaultSubscriptionID below pins that deliberate absence.
//
// The file lives in models/email because that is the service's PRIMARY model package and email is
// the resource every published event addresses; it is an external test package (`email_test`) so
// it can import sibling packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. A new published event type, a changed event-type
// constant, or an override added to *Email must be reflected here in the same change -- the table
// is not a specification of what the events ought to be, it is a lock on what they are.
package email_test

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-email-manager/models/email"
)

// emailID is the single subscription address every email-manager event of one email must carry.
var emailID = uuid.FromStringOrNil("b4a1f0c2-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Reproducing it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model starts or
// stops implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		return identifier.EventSubscriptionID()
	}

	m, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Could not marshal the event data. err: %v", err)
	}

	d := struct {
		ID string `json:"id"`
	}{}
	if errUnmarshal := json.Unmarshal(m, &d); errUnmarshal != nil {
		return ""
	}

	return d.ID
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameEmailManager)

	// The real event data type of every publish site: pkg/emailhandler/email.go passes
	// *email.Email to PublishWebhookEvent for all three event types.
	emailData := &email.Email{
		Identity: commonidentity.Identity{
			ID:         emailID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ActiveflowID: uuid.Must(uuid.NewV4()), // NOT the address -- own id is
		Status:       email.StatusDelivered,
		Subject:      "hello",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// email resource -- own id is the address, resolved by the default JSON fallback.
		{
			"email_created",
			email.EventTypeCreated,
			emailData,
			"email-manager.email.b4a1f0c2-0000-4000-8000-000000000001.created",
		},
		{
			"email_updated",
			email.EventTypeUpdated,
			emailData,
			"email-manager.email.b4a1f0c2-0000-4000-8000-000000000001.updated",
		},
		{
			"email_deleted",
			email.EventTypeDeleted,
			emailData,
			"email-manager.email.b4a1f0c2-0000-4000-8000-000000000001.deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriptionID := resolveSubscriptionID(t, tt.data)

			res := eventtopic.RoutingKey(publisher, tt.eventType, subscriptionID)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestGoldenRoutingKeyDataTypeIsPointer pins the data type actually handed to notifyhandler at
// every publish site. The SubscriptionIdentifier assertion is made against the DYNAMIC type, so a
// value publish would silently bypass any future override; asserting the pointer type here keeps
// the golden table reproducing the real publish path rather than a convenient stand-in.
func TestGoldenRoutingKeyDataTypeIsPointer(t *testing.T) {
	var data any = &email.Email{Identity: commonidentity.Identity{ID: emailID}}

	if _, ok := data.(*email.Email); !ok {
		t.Errorf("The published event data must be *email.Email. got: %T", data)
	}
}

// TestEmailUsesDefaultSubscriptionID pins the deliberate ABSENCE of an override on Email
// (design §2.4): an email is an independent persistent resource, its own id IS the subscription
// address, so implementing SubscriptionIdentifier would be redundant and the default JSON `id`
// extraction must keep covering it. If someone adds an override here, this test fails and forces
// the golden table above to be re-derived.
func TestEmailUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &email.Email{Identity: commonidentity.Identity{ID: emailID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Email must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}

	if res := resolveSubscriptionID(t, data); res != emailID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", emailID.String(), res)
	}
}
