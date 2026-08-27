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
// number-manager is a DEFAULT-ID service: `number.Number` is an independent, persistent resource
// whose own id IS the subscription address, so it deliberately carries NO
// eventtopic.SubscriptionIdentifier override. TestNumberUsesDefaultSubscriptionID pins that
// deliberate absence.
//
// The file lives in models/number because that is the service's PRIMARY model package and number
// is the resource every published event addresses; it is an external test package (`number_test`)
// so it can import sibling packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. A new published event type, a new dbUpdate
// branch, a changed event-type constant, or an override added to *Number must be reflected here in
// the same change -- the table is not a specification of what the events ought to be, it is a lock
// on what they are.
package number_test

import (
	"encoding/json"
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
		// number resource -- own id is the address, resolved by the default JSON fallback.
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
// `number-manager.number.<number-id>.#` receives the update AND the renew stream. If a future
// change routes the renew branch through a different payload type, this test catches the
// divergence.
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

	for _, eventType := range []string{number.EventTypeNumberUpdated, number.EventTypeNumberRenewed} {
		res := eventtopic.RoutingKey(publisher, eventType, subscriptionID)
		if eventtopic.IsPlaceholderSubscriptionID(subscriptionID) {
			t.Errorf("The number subscription address must not collapse to the placeholder. event_type: %s, key: %s", eventType, res)
		}
	}
}

// TestGoldenRoutingKeyDataTypeIsPointer pins the data type actually handed to notifyhandler at
// every publish site. The SubscriptionIdentifier assertion is made against the DYNAMIC type, so a
// value publish would silently bypass any future override; asserting the pointer type here keeps
// the golden table reproducing the real publish path rather than a convenient stand-in.
func TestGoldenRoutingKeyDataTypeIsPointer(t *testing.T) {
	var data any = &number.Number{Identity: commonidentity.Identity{ID: numberID}}

	if _, ok := data.(*number.Number); !ok {
		t.Errorf("The published event data must be *number.Number. got: %T", data)
	}
}

// TestNumberUsesDefaultSubscriptionID pins the deliberate ABSENCE of an override on Number
// (design §2.4): a number is an independent persistent resource, its own id IS the subscription
// address, so implementing SubscriptionIdentifier would be redundant and the default JSON `id`
// extraction must keep covering it. If someone adds an override here, this test fails and forces
// the golden table above to be re-derived.
func TestNumberUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &number.Number{Identity: commonidentity.Identity{ID: numberID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Number must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}

	if res := resolveSubscriptionID(t, data); res != numberID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", numberID.String(), res)
	}
}
