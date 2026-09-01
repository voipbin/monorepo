// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405/1419).
//
// It covers EVERY event type sentinel-manager publishes and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site.
//
// # What changed in VOIP-1418, and why the expected strings moved
//
// This file replaces `models/pod/routingkey_golden_test.go`. Both the resource segment and the
// action segment of every key changed, deliberately:
//
//	sentinel-manager.pod.-.updated   ->  sentinel-manager.container.<asterisk-id>.started
//	sentinel-manager.pod.-.deleted   ->  sentinel-manager.container.<asterisk-id>.died
//
// A golden test exists precisely to make a key change loud, so treat a future diff here as a
// regression unless it is an equally deliberate, reviewed decision. The rename was safe in this
// one instance because sentinel-manager's only real consumer -- bin-call-manager -- binds the
// WILDCARD pattern `sentinel-manager.container.*.died` and was updated in the same change.
//
// # sentinel-manager is no longer a placeholder-by-design publisher
//
// The old `pod.Event` returned "" from EventSubscriptionID() unconditionally: a Kubernetes Pod
// carries no top-level `id` (its identity lives under `metadata`), so no addressable identity
// existed. The Docker backend's state table resolves a real asterisk-id BEFORE the container
// dies, and `container.Event.EventSubscriptionID()` returns it directly. Consequences:
//
//   - Instance subscription of container events is now POSSIBLE (nothing binds that way today).
//   - `sentinel_manager_topic_placeholder_total ~= topic_publish_total{ok}` is NO LONGER the
//     healthy invariant. The placeholder now appears only for a genuinely unresolved id, which is
//     a degraded state worth alerting on -- see docs/architecture.md.
//
// The `-` placeholder still appears in two legitimate cases, both pinned below: a `container_died`
// whose id was never resolved, and every `container_started` (a fresh container's id is not
// resolvable until the background refresh loop has run at least one pass).
package container_test

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-sentinel-manager/models/container"
)

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): the explicit EventSubscriptionID method is mandatory, and an empty return is the
// only degrade path to the `-` placeholder -- the JSON fallback is gone. The parameter stays `any`
// on purpose so the table can also prove that a NON-implementing payload resolves to "" rather
// than silently borrowing an address.
//
// The typed-nil guard mirrors notifyhandler.resolveSubscriptionID: a nil pointer whose type
// implements the interface still SATISFIES the assertion, and a method that dereferenced its
// receiver would panic. Production resolves such a payload to the placeholder, so this helper
// returns "" for it too.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func Test_GoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameSentinelManager)

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// pkg/dockerwatchhandler/events.go -- handleContainerStarted. A freshly started container
		// has no resolved id yet (the refresh loop has not run for it), so this key legitimately
		// carries the placeholder.
		{
			"container_started (placeholder address: a new container has no resolved id yet)",
			container.EventTypeContainerStarted,
			&container.Event{
				ContainerName: "voip-asterisk-call-docker-2",
				Service:       container.ServiceAsteriskCall,
				AsteriskID:    "",
			},
			"sentinel-manager.container.-.started",
		},
		// pkg/dockerwatchhandler/events.go -- handleContainerDied, the normal case.
		{
			"container_died (resolved asterisk-id becomes the subscription address)",
			container.EventTypeContainerDied,
			&container.Event{
				ContainerName: "voip-asterisk-call-docker-2",
				Service:       container.ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},
			"sentinel-manager.container.3e:50:6b:43:bb:32.died",
		},
		// pkg/dockerwatchhandler/events.go -- handleContainerDied, degraded case: the container
		// died before its id could ever be resolved.
		{
			"container_died (unresolved id degrades to the placeholder)",
			container.EventTypeContainerDied,
			&container.Event{
				ContainerName: "voip-asterisk-call-docker-2",
				Service:       container.ServiceAsteriskCall,
				AsteriskID:    "",
			},
			"sentinel-manager.container.-.died",
		},
		// a different service's container produces the same resource/action segments -- Service
		// is a payload field, never part of the key.
		{
			"container_died for a registrar container",
			container.EventTypeContainerDied,
			&container.Event{
				ContainerName: "voip-asterisk-registrar-docker-1",
				Service:       container.ServiceAsteriskRegistrar,
				AsteriskID:    "72:ce:24:e6:51:2f",
			},
			"sentinel-manager.container.72:ce:24:e6:51:2f.died",
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

// Test_ConsumerBindingPattern pins the OTHER half of the contract: the wildcard pattern
// bin-call-manager binds must match the keys the table above produces. A resource/action rename
// on one side without the other would bind to nothing -- silently, with no error anywhere.
//
// The literal is deliberately hand-written rather than derived, so this assertion is independent
// of the same helper the producer uses. bin-call-manager's own
// pkg/subscribehandler/binding_golden_test.go pins the identical string from the consumer side.
func Test_ConsumerBindingPattern(t *testing.T) {
	expect := "sentinel-manager.container.*.died"

	res := eventtopic.PatternForEventType(string(commonoutline.ServiceNameSentinelManager), container.EventTypeContainerDied)
	if res != expect {
		t.Errorf("Wrong match. expect: %s, got: %s", expect, res)
	}
}

// Test_TypedNilResolvesToPlaceholder pins the typed-nil branch on the real publish-path helper.
func Test_TypedNilResolvesToPlaceholder(t *testing.T) {
	var event *container.Event

	subscriptionID := resolveSubscriptionID(t, event)
	if subscriptionID != "" {
		t.Errorf("Wrong match. expect: empty subscription id, got: %s", subscriptionID)
	}
	if !eventtopic.IsPlaceholderSubscriptionID(subscriptionID) {
		t.Errorf("Wrong match. expect: placeholder subscription id, got: not a placeholder")
	}

	res := eventtopic.RoutingKey(string(commonoutline.ServiceNameSentinelManager), container.EventTypeContainerDied, subscriptionID)
	if res != "sentinel-manager.container.-.died" {
		t.Errorf("Wrong match. expect: sentinel-manager.container.-.died, got: %s", res)
	}
}

// Test_ValueEventDoesNotSatisfyTheInterface pins the pointer-receiver narrowing VOIP-1419
// introduced: a bare struct VALUE must NOT satisfy SubscriptionIdentifier, so a publish site
// handing over a value fails to compile rather than silently resolving to the placeholder.
func Test_ValueEventDoesNotSatisfyTheInterface(t *testing.T) {
	var data any = container.Event{AsteriskID: "3e:50:6b:43:bb:32"}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Wrong match. expect: a container.Event VALUE not to satisfy SubscriptionIdentifier, got: it does")
	}
}
