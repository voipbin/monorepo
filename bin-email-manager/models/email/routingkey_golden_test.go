// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type email-manager publishes today (`email_created`, `email_updated`,
// `email_deleted` -- all three from pkg/emailhandler/email.go) and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. The defect class it
// guards against is "the right key shape carrying the wrong id space": a key whose third segment
// is not the address subscribers can bind to in advance produces well-formed keys that no
// instance binding ever matches, and no runtime metric can detect it. Design doc §2.4 / §4.
//
// email-manager is an OWN-ID service: `email.Email` is an independent, persistent resource whose
// own id IS the subscription address, stated by the mandatory `EventSubscriptionID()`
// promoted from the embedded commonidentity.Identity (VOIP-1419 -- every published type implements
// eventtopic.SubscriptionIdentifier; an empty return degrades to the `-` placeholder).
//
// The file lives in models/email because that is the service's PRIMARY model package and email is
// the resource every published event addresses; it is an external test package (`email_test`) so
// it can import sibling packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. A new published event type, a changed event-type
// constant, or a changed `EventSubscriptionID()` on *Email must be reflected here in the same
// change -- the table is not a specification of what the events ought to be, it is a lock on what
// they are.
package email_test

import (
	"reflect"
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
// (VOIP-1419): the payload's mandatory `EventSubscriptionID()` method is the ONLY source of the
// subscription-id segment -- there is no JSON fallback anymore. Reproducing it here rather than
// reaching into notifyhandler internals is deliberate -- the golden table must fail when a
// model's method starts returning a different id space.
//
// The parameter stays `any` (not the interface): a non-implementing payload returns "" (the `-`
// placeholder), which is what production's narrowed signature makes uncompilable but the table
// must still be able to express. The typed-nil guard mirrors production: a nil pointer whose type
// implements the interface still satisfies the assertion, and every real implementation
// dereferences its receiver -- calling the method would panic, so it degrades to "" instead.
func resolveSubscriptionID(data any) string {
	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
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
		// email resource -- own id is the address, returned through the promoted Identity default.
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
			subscriptionID := resolveSubscriptionID(tt.data)

			res := eventtopic.RoutingKey(publisher, tt.eventType, subscriptionID)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}
