// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type tag-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. tag-manager has a
// single resource namespace (`tag`) and a single published type (`*tag.Tag`), whose
// `EventSubscriptionID()`, promoted from the embedded commonidentity.Identity, returns the
// tag's own id (VOIP-1419): a tag is an independent
// persistent resource addressed by its own id (design §2.4). The method is mandatory -- an
// empty return degrades to the `-` placeholder -- and the table exists to keep the address
// stable: a method change that relocated every subscriber's binding address would keep the
// keys well-formed the whole time, which no runtime metric detects.
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
// (VOIP-1419): the `EventSubscriptionID()` method (for tag.Tag, the own-id default promoted
// from the embedded commonidentity.Identity) is the whole mechanism -- there is no
// JSON fallback. A payload that does not implement the interface, or a typed-nil pointer whose
// method would panic on dereference, resolves to "" and thereby to the `-` placeholder segment.
// Keeping the reproduction here rather than reaching into notifyhandler internals is deliberate --
// the golden table must fail when a model's method starts returning a different address.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
		// interface still SATISFIES the assertion, and every real implementation dereferences its
		// receiver -- calling the method would panic. Production resolves such a payload to the
		// `-` placeholder, so "" is returned here instead of calling the method.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Ptr || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
	}

	return ""
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
