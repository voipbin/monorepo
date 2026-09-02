// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type registrar-manager publishes today, across both resource namespaces
// (trunk / extension), and asserts the exact key that notifyhandler generates for the real event
// data type of each publish site. Since VOIP-1419 the subscription-id segment is resolved by the
// mandatory `EventSubscriptionID()` contract -- for both of this service's types via the
// own-id default promoted from the embedded commonidentity.Identity (an empty return
// degrades to the `-` placeholder); there is no JSON fallback. Both of this service's types
// return their own id. The table is what proves that -- a method silently changed to return a
// different field changes these keys, and no runtime metric would detect it because the keys
// would still be well formed.
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
// (VOIP-1419): the mandatory `EventSubscriptionID()` contract (satisfied here by the method
// promoted from the embedded commonidentity.Identity), with an empty result (or a
// non-implementing / typed-nil payload) degrading to "" -- the `-` placeholder segment. Keeping
// it here rather than reaching into notifyhandler internals is deliberate -- the golden table
// must fail when a model's method starts returning a different address, which is exactly what
// this reproduction detects. The parameter stays `any` to match the pre-narrowing publish
// signature the table exercises.
func resolveSubscriptionID(data any) string {
	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the interface
	// still SATISFIES the assertion, and every real implementation dereferences its receiver --
	// calling the method would panic. Such a payload resolves to the `-` placeholder.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Pointer && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
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
		// trunk resource -- own id is the address, returned through the promoted Identity default.
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

		// extension resource -- own id is the address, returned through the promoted Identity default.
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
			subscriptionID := resolveSubscriptionID(tt.data)

			res := eventtopic.RoutingKey(publisher, tt.eventType, subscriptionID)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestRegistrarExplicitMethodResolvesOwnID pins that BOTH published types satisfy the
// mandatory contract, via the EventSubscriptionID promoted from the embedded
// commonidentity.Identity (a non-implementing payload would resolve to "" here, breaking the
// expectation) AND that the method yields the resource's own id -- the same value the retired
// JSON `id` fallback extracted, so the key strings above are unchanged by VOIP-1419.
func TestRegistrarExplicitMethodResolvesOwnID(t *testing.T) {
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
			res := resolveSubscriptionID(tt.data)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}
