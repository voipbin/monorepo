// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type direct-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. The primary defect
// class it guards against is "the right key shape carrying the wrong id space": an event published
// under an id that is not the resource's subscription address produces well-formed keys that no
// instance binding ever matches, and no runtime metric can detect it. Design doc §2.4 / §4.
//
// direct-manager is a default-fallback service: every published payload is a `*direct.Direct`
// (all three publish sites funnel through directhandler.publishEvent, which is typed to
// `*direct.Direct`), whose own id IS the subscription address, so NO type in this service carries
// an eventtopic.SubscriptionIdentifier override. That absence is asserted explicitly below.
//
// The address choice worth naming: a Direct carries a `resource_id` pointing at the agent, queue,
// conference, ... it fronts. That foreign id is NOT the address -- a consumer following one direct
// binds `direct-manager.direct.<direct-id>.#`, and events about the fronted resource come from that
// resource's own publisher.
//
// The file lives in models/direct because direct is the only publishing model package of the
// service; it is an external test package so it can import sibling packages without any
// import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. All three constants in models/direct/event.go are
// LIVE (handler.go:75/182/239) -- there is no dead constant to exclude here. When a publish path is
// added or removed, update this table in the same change.
package direct_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-direct-manager/models/direct"
)

// directID is the single subscription address every direct-manager event of one direct must carry.
var directID = uuid.FromStringOrNil("b90f6a38-0000-4000-8000-000000000001")

// resourceID is the id of the resource the direct fronts -- deliberately different from directID so
// the table fails if the address ever slips to it.
var resourceID = uuid.FromStringOrNil("b90f6a38-0000-4000-8000-000000000002")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Keeping it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model STARTS or STOPS
// implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly; if the two ever
// diverge, this table stops reproducing what the publish path actually generates.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler.resolveSubscriptionOverride: a nil pointer whose
		// type implements the interface still SATISFIES the assertion, and every real implementation
		// dereferences its receiver -- calling the method would panic. Production reports "no
		// override" for such a payload, so this guard falls through to the JSON half below rather
		// than returning early; `null` carries no top-level `id` either, so both halves agree on the
		// `-` placeholder.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Ptr || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
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
	publisher := string(commonoutline.ServiceNameDirectManager)

	directData := &direct.Direct{
		Identity: commonidentity.Identity{
			ID:         directID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ResourceType: direct.ResourceTypeAgent,
		ResourceID:   resourceID,
		Hash:         "direct.testhash",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// direct resource -- own id is the address, resolved by the default JSON fallback.
		{
			"direct_created",
			direct.EventTypeDirectCreated,
			directData,
			"direct-manager.direct.b90f6a38-0000-4000-8000-000000000001.created",
		},
		{
			"direct_deleted",
			direct.EventTypeDirectDeleted,
			directData,
			"direct-manager.direct.b90f6a38-0000-4000-8000-000000000001.deleted",
		},
		// A regenerate keeps the SAME direct id and only rotates the hash, so the address is stable
		// across regeneration -- an instance binding made before the rotation keeps working.
		{
			"direct_regenerated",
			direct.EventTypeDirectRegenerated,
			directData,
			"direct-manager.direct.b90f6a38-0000-4000-8000-000000000001.regenerated",
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

// TestGoldenRoutingKeysUseOwnID pins the property the table exists to protect: the address is the
// direct's OWN id, taken from the marshaled payload's top-level `id` -- never the `resource_id` of
// the resource it fronts, and never the placeholder.
func TestGoldenRoutingKeysUseOwnID(t *testing.T) {
	data := &direct.Direct{
		Identity:     commonidentity.Identity{ID: directID, CustomerID: uuid.Must(uuid.NewV4())},
		ResourceType: direct.ResourceTypeQueue,
		ResourceID:   resourceID,
	}

	res := resolveSubscriptionID(t, data)
	if res != directID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", directID.String(), res)
	}
	if res == resourceID.String() {
		t.Errorf("Direct must not be addressed by the id of the resource it fronts. got: %s", res)
	}
	if eventtopic.IsPlaceholderSubscriptionID(res) {
		t.Errorf("Direct must never resolve to the placeholder address. got: %s", res)
	}
}

// TestDirectUsesDefaultSubscriptionID pins the deliberate absence of an override on Direct: its own
// id IS the address, so implementing the interface would be redundant and the default JSON `id`
// extraction must keep covering it. direct-manager has no override type at all (design §2.4).
func TestDirectUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &direct.Direct{Identity: commonidentity.Identity{ID: directID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Direct must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}
