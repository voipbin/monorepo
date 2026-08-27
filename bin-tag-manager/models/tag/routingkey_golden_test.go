// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type tag-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. tag-manager has a
// single resource namespace (`tag`) and a single published type (`*tag.Tag`), and it carries NO
// subscription-id override: a tag is an independent persistent resource addressed by its own id
// (design §2.4), so the default JSON top-level `id` fallback is the whole resolution. The table
// exists to keep it that way -- an override silently added to *Tag would relocate every
// subscriber's binding address while the keys stayed well-formed, which no runtime metric detects.
//
// The file lives in models/tag, the service's only (and therefore PRIMARY) model package; it is
// an external test package for consistency with the other services' golden tables, whose ones
// must span sibling model packages.
//
// MAINTENANCE: this table pins CURRENT behavior. All three constants are live -- publish sites are
// pkg/taghandler/db.go:103 (created), :66 (updated), :127 (deleted, also reached by the
// customer_deleted cascade in pkg/subscribehandler).
package tag_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-tag-manager/models/tag"
)

// tagID is the subscription address of every tag-manager event.
var tagID = uuid.FromStringOrNil("7b2e4c60-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Keeping it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model STARTS or
// STOPS implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly.
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
	publisher := string(commonoutline.ServiceNameTagManager)

	tagData := &tag.Tag{
		Identity: commonidentity.Identity{
			ID:         tagID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Name:   "vip",
		Detail: "vip customers",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		{
			"tag_created",
			tag.EventTypeTagCreated,
			tagData,
			"tag-manager.tag.7b2e4c60-0000-4000-8000-000000000001.created",
		},
		{
			"tag_updated",
			tag.EventTypeTagUpdated,
			tagData,
			"tag-manager.tag.7b2e4c60-0000-4000-8000-000000000001.updated",
		},
		{
			"tag_deleted",
			tag.EventTypeTagDeleted,
			tagData,
			"tag-manager.tag.7b2e4c60-0000-4000-8000-000000000001.deleted",
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

// TestTagUsesDefaultSubscriptionID pins the deliberate ABSENCE of a subscription-id override on
// *tag.Tag (design §2.4: tag addresses by its own id). Adding one would silently relocate every
// subscriber's binding address, and the keys would stay well-formed the whole time.
func TestTagUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &tag.Tag{Identity: commonidentity.Identity{ID: tagID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Tag must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}
