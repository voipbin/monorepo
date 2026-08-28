// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type registrar-manager publishes today, across both resource namespaces
// (trunk / extension), and asserts the exact key that notifyhandler generates for the real event
// data type of each publish site. registrar-manager is a default-id service: it declares NO
// SubscriptionIdentifier override, so every key's third segment comes from the JSON `id` fallback
// (design §2.4). The table is what proves that -- an override silently added later (or an `id`
// json tag silently renamed) changes these keys, and no runtime metric would detect it because
// the keys would still be well formed.
//
// The file lives in models/trunk (the service's designated PRIMARY model package for this table,
// chosen for practical anchoring rather than strict aggregate semantics -- trunk and extension
// are peer resources, not parent/child); it is an external test package so it can import the
// sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior, not what the events ought to be.
//
// Publish-site inventory pinned here (all four NotifyHandler construction sites of this service
// carry notifyhandler.WithGlobalTopicPublish(), so every one of these paths reaches the topic
// exchange):
//   - pkg/trunkhandler/trunk.go:106/:201/:231 -- trunk_created / _updated / _deleted, *trunk.Trunk
//   - pkg/extensionhandler/extension.go:165/:274/:337 -- extension_created / _updated / _deleted,
//     *extension.Extension
//   - cmd/registrar-control/domain_migrate.go:591 -- extension_updated published directly from cmd
//     code by the domain migration batch, *extension.Extension. This is why registrar-control's
//     THREE constructors all get the option (design §1.1, publisher-stream completeness): the
//     migration batch is a live publisher, and its key must be the same shape as the one
//     registrar-manager produces for the same event type. The row below covers it.
package trunk_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/trunk"
)

// trunkID / extensionID are the subscription addresses of the two namespaces. Unlike the
// parent-axis services (transcribe, conference, ...), registrar-manager's two resources are
// INDEPENDENT: a trunk and an extension are separate top-level resources, each addressed by its
// own id, so there is no shared address to converge on. Two distinct ids here make that explicit.
var (
	trunkID     = uuid.FromStringOrNil("a71b3e40-0000-4000-8000-000000000001")
	extensionID = uuid.FromStringOrNil("a71b3e40-0000-4000-8000-000000000002")
)

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
	publisher := string(commonoutline.ServiceNameRegistrarManager)

	trunkData := &trunk.Trunk{
		Identity: commonidentity.Identity{
			ID:         trunkID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Name:       "test trunk",
		DomainName: "test.trunk.voipbin.net",
	}

	extensionData := &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         extensionID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Extension:  "1001",
		DomainName: "abcd.reg.voipbin.net",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// trunk resource -- own id is the address, resolved by the default JSON fallback.
		{
			"trunk_created",
			trunk.EventTypeTrunkCreated,
			trunkData,
			"registrar-manager.trunk.a71b3e40-0000-4000-8000-000000000001.created",
		},
		{
			"trunk_updated",
			trunk.EventTypeTrunkUpdated,
			trunkData,
			"registrar-manager.trunk.a71b3e40-0000-4000-8000-000000000001.updated",
		},
		{
			"trunk_deleted",
			trunk.EventTypeTrunkDeleted,
			trunkData,
			"registrar-manager.trunk.a71b3e40-0000-4000-8000-000000000001.deleted",
		},

		// extension resource -- own id is the address, resolved by the default JSON fallback.
		{
			"extension_created",
			extension.EventTypeExtensionCreated,
			extensionData,
			"registrar-manager.extension.a71b3e40-0000-4000-8000-000000000002.created",
		},
		{
			"extension_updated",
			extension.EventTypeExtensionUpdated,
			extensionData,
			"registrar-manager.extension.a71b3e40-0000-4000-8000-000000000002.updated",
		},
		{
			"extension_deleted",
			extension.EventTypeExtensionDeleted,
			extensionData,
			"registrar-manager.extension.a71b3e40-0000-4000-8000-000000000002.deleted",
		},

		// Same event type, published from cmd/registrar-control/domain_migrate.go:591 instead of
		// the extension handler. Same data type, same key -- a consumer following an extension
		// cannot tell the migration batch apart from a normal update, which is the intended
		// contract. This row is why registrar-control's three constructors must carry the option.
		{
			"extension_updated (domain migration batch, cmd publish path)",
			extension.EventTypeExtensionUpdated,
			extensionData,
			"registrar-manager.extension.a71b3e40-0000-4000-8000-000000000002.updated",
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

// TestRegistrarModelsUseDefaultSubscriptionID pins the deliberate ABSENCE of an override on both
// published types (design §2.4): each resource's own id IS its address, so implementing
// SubscriptionIdentifier would be redundant, and the default JSON `id` extraction must keep
// covering them. This is the assertion that fails if somebody adds an `EventSubscriptionID()`
// method to *Trunk or *Extension without revisiting the table above.
func TestRegistrarModelsUseDefaultSubscriptionID(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{"trunk", &trunk.Trunk{Identity: commonidentity.Identity{ID: trunkID}}},
		{"extension", &extension.Extension{Identity: commonidentity.Identity{ID: extensionID}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := tt.data.(eventtopic.SubscriptionIdentifier); ok {
				t.Errorf("%s must not implement SubscriptionIdentifier. its own id is the subscription address.", tt.name)
			}
		})
	}
}

// TestRegistrarDefaultFallbackResolvesOwnID proves the other half of the default path: with no
// override in play, the resolved address is exactly the marshaled `id`. Together with the test
// above, this pins BOTH "no override exists" and "the fallback yields the own id" -- either one
// alone would still pass if the `id` json tag were renamed.
func TestRegistrarDefaultFallbackResolvesOwnID(t *testing.T) {
	tests := []struct {
		name   string
		data   any
		expect string
	}{
		{"trunk", &trunk.Trunk{Identity: commonidentity.Identity{ID: trunkID}}, trunkID.String()},
		{"extension", &extension.Extension{Identity: commonidentity.Identity{ID: extensionID}}, extensionID.String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveSubscriptionID(t, tt.data)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}
