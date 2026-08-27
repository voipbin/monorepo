// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type outdial-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. outdial-manager is a
// default-id service: it declares NO SubscriptionIdentifier override, so every key's third
// segment comes from the JSON `id` fallback (design §2.4). The table is what proves that -- an
// override silently added later (or an `id` json tag silently renamed) changes these keys, and
// no runtime metric would detect it because the keys would still be well formed.
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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-outdial-manager/models/outdial"
)

// outdialID is the subscription address every outdial-manager event carries: the outdial's own
// id, resolved through the default JSON fallback.
var outdialID = uuid.FromStringOrNil("6c14ab90-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Keeping it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model starts (or
// stops) implementing the interface, which is exactly what this two-step reproduction detects.
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
	publisher := string(commonoutline.ServiceNameOutdialManager)

	// The event data type of every outdial-manager publish site is *outdial.Outdial:
	// pkg/outdialhandler/outdial.go:62 (created), :85 (deleted), :184/:214/:244 (updated), all
	// passing the `res` returned by the handler -- a pointer, which is what the interface
	// assertion in resolveSubscriptionID would need to match if an override ever appeared.
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
		// outdial resource -- own id is the address, resolved by the default JSON fallback.
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

// TestOutdialUsesDefaultSubscriptionID pins the deliberate ABSENCE of an override on Outdial
// (design §2.4): its own id IS the address, so implementing SubscriptionIdentifier would be
// redundant, and the default JSON `id` extraction must keep covering it. This is the assertion
// that fails if somebody adds an `EventSubscriptionID()` method to *Outdial without revisiting
// the table above.
func TestOutdialUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &outdial.Outdial{Identity: commonidentity.Identity{ID: outdialID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Outdial must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}

// TestOutdialDefaultFallbackResolvesOwnID proves the other half of the default path: with no
// override in play, the resolved address is exactly the marshaled `id`. Together with the test
// above, this pins BOTH "no override exists" and "the fallback yields the own id" -- either one
// alone would still pass if the `id` json tag were renamed.
func TestOutdialDefaultFallbackResolvesOwnID(t *testing.T) {
	data := &outdial.Outdial{Identity: commonidentity.Identity{ID: outdialID}}

	res := resolveSubscriptionID(t, data)
	if res != outdialID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", outdialID.String(), res)
	}
}
