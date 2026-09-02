// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type number-manager publishes today:
//   - number_created (pkg/numberhandler/db.go:78, PublishWebhookEvent)
//   - number_deleted (pkg/numberhandler/db.go:103, PublishWebhookEvent)
//   - number_updated / number_renewed -- the two DYNAMIC branches of dbUpdate
//     (pkg/numberhandler/db.go:145-171). dbUpdate takes the event type as a parameter and switches
//     on it: `number_renewed` goes out through PublishEvent (renewal is internal bookkeeping, not a
//     customer webhook) while everything else -- today only `number_updated`, from
//     numberhandler/number.go:322,344 -- goes out through PublishWebhookEvent. Both publish paths
//     funnel through the same notifyhandler routing-key generation with the same *number.Number
//     payload, so BOTH branches are enumerated below; a table that only covered the webhook branch
//     would leave the renew path unpinned.
//
// The defect class this guards against is "the right key shape carrying the wrong id space": a key
// whose third segment is not the address subscribers can bind to in advance produces well-formed
// keys that no instance binding ever matches, and no runtime metric can detect it. Design doc
// §2.4 / §4.
//
// number-manager is an OWN-ID service: `number.Number` is an independent, persistent resource
// whose own id IS the subscription address, stated by the EventSubscriptionID promoted from
// the embedded commonidentity.Identity
// (VOIP-1419 -- the contract is mandatory and compiler-enforced; an empty return degrades to
// the `-` placeholder). The compile-time interface assertion lives in models/number/number_test.go.
//
// The file lives in models/number because that is the service's PRIMARY model package and number
// is the resource every published event addresses; it is an external test package (`number_test`)
// so it can import sibling packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. A new published event type, a new dbUpdate
// branch, a changed event-type constant, or a changed EventSubscriptionID implementation on
// *Number must be reflected here in the same change -- the table is not a specification of what
// the events ought to be, it is a lock on what they are.
package number_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-number-manager/models/number"
)

// numberID is the single subscription address every number-manager event of one number must carry,
// on BOTH publish paths (PublishWebhookEvent and PublishEvent).
var numberID = uuid.FromStringOrNil("e5c93a71-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): every published event data type implements eventtopic.SubscriptionIdentifier --
// the compiler enforces it at the publish call site -- and the method's return value IS the
// subscription address, with an empty return degrading to the `-` placeholder downstream.
// Reproducing it here rather than reaching into notifyhandler internals is deliberate -- the
// golden table must fail when a model's address derivation changes.
//
// The parameter stays `any` on purpose: non-implementing data (unrepresentable on the production
// path, where the signature rejects it at compile time) resolves to "" for the same placeholder.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the interface
	// still SATISFIES the assertion, and every real implementation dereferences its receiver --
	// calling the method would panic. Production resolves such a payload to the `-` placeholder.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Pointer && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameNumberManager)

	// The real event data type of every publish site: pkg/numberhandler/db.go passes
	// *number.Number to both PublishWebhookEvent and PublishEvent.
	numberData := &number.Number{
		Identity: commonidentity.Identity{
			ID:         numberID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Number:        "+821100000001",
		Type:          number.TypeNormal,
		CallFlowID:    uuid.Must(uuid.NewV4()), // NOT the address -- own id is
		MessageFlowID: uuid.Must(uuid.NewV4()), // NOT the address -- own id is
		ProviderName:  number.ProviderNameTelnyx,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// number resource -- own id is the address, returned through the promoted Identity default.
		{
			"number_created",
			number.EventTypeNumberCreated,
			numberData,
			"number-manager.number.e5c93a71-0000-4000-8000-000000000001.created",
		},
		{
			"number_deleted",
			number.EventTypeNumberDeleted,
			numberData,
			"number-manager.number.e5c93a71-0000-4000-8000-000000000001.deleted",
		},

		// dbUpdate dynamic branch 1 -- default case, PublishWebhookEvent path.
		{
			"number_updated (dbUpdate default branch, PublishWebhookEvent)",
			number.EventTypeNumberUpdated,
			numberData,
			"number-manager.number.e5c93a71-0000-4000-8000-000000000001.updated",
		},

		// dbUpdate dynamic branch 2 -- renewed case, PublishEvent path. Same payload, same
		// address, different publish call: the routing key must be indistinguishable in shape.
		{
			"number_renewed (dbUpdate renew branch, PublishEvent)",
			number.EventTypeNumberRenewed,
			numberData,
			"number-manager.number.e5c93a71-0000-4000-8000-000000000001.renewed",
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

// TestGoldenRoutingKeysDbUpdateBranchesShareOneAddress pins the property the two dbUpdate rows
// above exist to protect: the publish path taken (PublishEvent vs PublishWebhookEvent) changes
// nothing about the subscription address, so one instance binding
// `number-manager.number.<number-id>.#` receives the update AND the renew stream.
//
// Scope: this pins the KEY-DERIVATION property for the shared *number.Number payload -- that both
// event types resolve to the same address and land under one binding. It does NOT observe the
// publish sites, so a future change that routes the renew branch through a different payload type
// is caught by the golden table above (whose rows name the real publish sites), not here.
func TestGoldenRoutingKeysDbUpdateBranchesShareOneAddress(t *testing.T) {
	publisher := string(commonoutline.ServiceNameNumberManager)
	numberData := &number.Number{Identity: commonidentity.Identity{ID: numberID}}

	subscriptionID := resolveSubscriptionID(t, numberData)
	if subscriptionID != numberID.String() {
		t.Fatalf("Wrong match. expect: %s, got: %s", numberID.String(), subscriptionID)
	}

	pattern := eventtopic.PatternInstance(publisher, "number", subscriptionID)
	if pattern != "number-manager.number.e5c93a71-0000-4000-8000-000000000001.#" {
		t.Errorf("Wrong match. expect: number-manager.number.e5c93a71-0000-4000-8000-000000000001.#, got: %s", pattern)
	}

	// Every branch's key must fall UNDER that one binding (same prefix, only the trailing event-type
	// segment varies), and the two branches must still be distinguishable by event type -- one
	// address, two streams.
	prefix := strings.TrimSuffix(pattern, "#")
	keys := map[string]string{}
	for _, eventType := range []string{number.EventTypeNumberUpdated, number.EventTypeNumberRenewed} {
		res := eventtopic.RoutingKey(publisher, eventType, subscriptionID)
		if !strings.HasPrefix(res, prefix) {
			t.Errorf("The routing key must be matched by the instance binding. pattern: %s, event_type: %s, key: %s", pattern, eventType, res)
		}
		keys[eventType] = res
	}

	if keys[number.EventTypeNumberUpdated] == keys[number.EventTypeNumberRenewed] {
		t.Errorf("The two dbUpdate branches must stay distinguishable by event type. key: %s", keys[number.EventTypeNumberUpdated])
	}
}
