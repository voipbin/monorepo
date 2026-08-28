// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type outdial-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. outdial-manager is
// an OWN-ID service: `outdial.Outdial` satisfies eventtopic.SubscriptionIdentifier through the
// method promoted from its embedded commonidentity.Identity
// (the contract is mandatory since VOIP-1419; an empty return degrades to the `-` placeholder) and returns the
// resource's own id. The table is what pins that -- an EventSubscriptionID implementation that
// starts returning a different id space changes these keys, and no runtime metric would detect
// it because the keys would still be well formed.
//
// The file lives in models/outdial because outdial is the only resource this service publishes
// and every published payload addresses it; it is an external test package so it can import the
// sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior, not what the events ought to be.
//
// The `outdialtarget_created` / `_updated` / `_deleted` constants in models/outdialtarget are
// DEAD -- declared, but never passed to any Publish* call anywhere in the service (verified by
// grepping `notifyHandler.` across pkg/). They are deliberately absent from this table: pinning a
// key for an event that is never published would assert a fiction. If outdialtargethandler ever
// starts publishing them, add their rows here in the SAME change -- and note that the resource
// segment would then be `outdialtarget`, a namespace distinct from `outdial` (segment boundaries
// are whole, so `outdial-manager.outdial.#` never matches an `outdialtarget` key).
package outdial_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-outdial-manager/models/outdial"
)

// outdialID is the subscription address every outdial-manager event carries: the outdial's own
// id, returned through the EventSubscriptionID promoted from the embedded commonidentity.Identity.
var outdialID = uuid.FromStringOrNil("6c14ab90-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): every published type satisfies the mandatory EventSubscriptionID contract (here
// via the method promoted from the embedded commonidentity.Identity);
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
	publisher := string(commonoutline.ServiceNameOutdialManager)

	// The event data type of every outdial-manager publish site is *outdial.Outdial:
	// pkg/outdialhandler/outdial.go:62 (created), :85 (deleted), :184/:214/:244 (updated), all
	// passing the `res` returned by the handler -- a pointer, which is the dynamic type the
	// interface check in resolveSubscriptionID matches.
	outdialData := &outdial.Outdial{
		Identity: commonidentity.Identity{
			ID:         outdialID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		CampaignID: uuid.Must(uuid.NewV4()),
		Name:       "test outdial",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// outdial resource -- own id is the address, returned through the promoted Identity default.
		{
			"outdial_created",
			outdial.EventTypeOutdialCreated,
			outdialData,
			"outdial-manager.outdial.6c14ab90-0000-4000-8000-000000000001.created",
		},
		{
			"outdial_deleted",
			outdial.EventTypeOutdialDeleted,
			outdialData,
			"outdial-manager.outdial.6c14ab90-0000-4000-8000-000000000001.deleted",
		},
		// One event type, three publish sites (Update / UpdateCampaignID / UpdateData); they all
		// publish the same type with the same data type, so one row pins all three.
		{
			"outdial_updated",
			outdial.EventTypeOutdialUpdated,
			outdialData,
			"outdial-manager.outdial.6c14ab90-0000-4000-8000-000000000001.updated",
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

// TestGoldenRoutingKeysUseOwnID pins the property the table exists to protect: the address is
// the outdial's OWN id, returned through the promoted Identity default -- never the
// campaign it belongs to, and never the placeholder. (The mutation-checked per-field
// address-choice test lives in the model's own package next to the method.)
func TestGoldenRoutingKeysUseOwnID(t *testing.T) {
	campaignID := uuid.Must(uuid.NewV4())
	data := &outdial.Outdial{
		Identity:   commonidentity.Identity{ID: outdialID, CustomerID: uuid.Must(uuid.NewV4())},
		CampaignID: campaignID,
	}

	res := resolveSubscriptionID(t, data)
	if res != outdialID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", outdialID.String(), res)
	}
	if res == campaignID.String() {
		t.Errorf("Outdial must not be addressed by its campaign id. got: %s", res)
	}
	if eventtopic.IsPlaceholderSubscriptionID(res) {
		t.Errorf("Outdial must never resolve to the placeholder address. got: %s", res)
	}
}
