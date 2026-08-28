// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type direct-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. The primary defect
// class it guards against is "the right key shape carrying the wrong id space": an event published
// under an id that is not the resource's subscription address produces well-formed keys that no
// instance binding ever matches, and no runtime metric can detect it. Design doc §2.4 / §4.
//
// direct-manager is an OWN-ID service: every published payload is a `*direct.Direct`
// (all three publish sites funnel through directhandler.publishEvent, which is typed to
// `*direct.Direct`), which implements eventtopic.SubscriptionIdentifier explicitly (mandatory
// since VOIP-1419; an empty return degrades to the `-` placeholder) and returns the resource's
// own id.
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
// (VOIP-1419): every published type carries an explicit, mandatory EventSubscriptionID method;
// the method's return is the address, and an empty return (or a non-implementing / nil payload)
// degrades to the `-` placeholder. Reproducing it here rather than reaching into notifyhandler
// internals is deliberate -- the golden table must fail when a model's method starts returning a
// different id space.
//
// The parameter stays `any` on purpose: a non-implementing payload resolves to "" (placeholder)
// rather than failing to compile, matching the production helper's degrade path.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
		// interface still SATISFIES the assertion, and every real implementation dereferences its
		// receiver -- calling the method would panic. Production resolves such a payload to the
		// `-` placeholder instead.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Ptr || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
	}

	return ""
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
		// direct resource -- own id is the address, returned by the explicit EventSubscriptionID.
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
// direct's OWN id, returned by the explicit EventSubscriptionID method -- never the `resource_id`
// of the resource it fronts, and never the placeholder.
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
