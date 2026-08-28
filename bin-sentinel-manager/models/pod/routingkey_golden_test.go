// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405/1419).
//
// It covers EVERY event type sentinel-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site.
//
// sentinel-manager is the ONE documented placeholder-by-design publisher (design §2.4 / §5).
// pkg/monitoringhandler/run.go publishes `pod.Event`, a wrapper that anonymously embeds the
// informer's `*corev1.Pod` by POINTER — the embed inlines on marshal and Pod has no MarshalJSON,
// so the payload bytes are identical to the bare pod every fanout consumer has always received.
// Since VOIP-1419 the subscription address is explicit: `Event.EventSubscriptionID()` returns ""
// by design (a Pod's identity lives under `metadata`; there is no top-level `id` to address), and
// eventtopic.RoutingKey collapses the empty segment to the `-` placeholder. The keys are
// therefore `sentinel-manager.pod.-.updated` / `.deleted` for EVERY pod, and instance
// subscription of pod events is not supported. The runbook consequence is recorded in
// docs/architecture.md: `sentinel_manager_topic_placeholder_total ~= topic_publish_total{ok}` is
// the HEALTHY invariant here, not an alert condition -- but the ratio still detects publish
// regressions.
//
// The `pod.Event` wrapper is the ONE sanctioned wrapping of the pod payload: it preserves the
// marshaled shape byte-for-byte and exists only to carry the explicit subscription address. Do
// NOT replace it with a shape-CHANGING wrapper (e.g. one publishing the pod under a named field,
// or the value-embedding models/pod.Pod) — that would alter the payload every existing fanout
// consumer parses. And do NOT attach methods to the OLD `pod.Pod` wrapper or expect them on
// `*corev1.Pod` itself: pod.Pod anonymously embeds corev1.Pod, so a method on either would
// promote onto the other's method set — the same wrapper-promotion hazard design §2.4 calls out
// for customer.CustomerCreatedEvent.
//
// MAINTENANCE: `pod_added` (EventTypePodAdded) is a DEAD constant -- the informer's AddFunc is an
// intentional no-op (Kubernetes replays existing pods as Add during the initial list), so nothing
// publishes it. It is deliberately absent from the table; if AddFunc ever starts publishing, add
// its row in the same change.
package pod_test

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-sentinel-manager/models/pod"
)

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): the explicit EventSubscriptionID method is mandatory, and an empty return is the
// only degrade path to the `-` placeholder — the JSON fallback is gone. The parameter stays `any`
// on purpose: this table must also prove that NON-implementing payloads (the bare `*corev1.Pod`
// below) resolve to "" rather than silently borrowing an address, which an interface-typed
// parameter could not even express.
//
// The typed-nil guard mirrors notifyhandler.resolveSubscriptionID: a nil pointer whose type
// implements the interface still SATISFIES the assertion, and calling a method that dereferences
// its receiver would panic. Production resolves such a payload to the placeholder, so this helper
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

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameSentinelManager)

	// The exact value monitoringhandler hands to PublishEvent: the informer's own object wrapped
	// in pod.Event. Populated with a realistic uid/name to make the point that NONE of it reaches
	// the key.
	podData := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       "3f2b0d18-0000-4000-8000-000000000001",
			Name:      "asterisk-call-7c9d5f8b6d-abcde",
			Namespace: "voip",
			Labels:    map[string]string{"app": "asterisk-call"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// pkg/monitoringhandler/run.go:105 -- informer UpdateFunc.
		{
			"pod_updated (placeholder address by design)",
			pod.EventTypePodUpdated,
			&pod.Event{Pod: podData},
			"sentinel-manager.pod.-.updated",
		},
		// pkg/monitoringhandler/run.go:117 -- informer DeleteFunc.
		{
			"pod_deleted (placeholder address by design)",
			pod.EventTypePodDeleted,
			&pod.Event{Pod: podData},
			"sentinel-manager.pod.-.deleted",
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

// TestPodPayloadHasNoSubscriptionAddress pins the two independent producers of the placeholder:
// the bare `*corev1.Pod` (an external type that CANNOT carry a subscription address) resolves to
// "" because it does not implement the interface at all, and the published `pod.Event` wrapper
// resolves to "" because its explicit method says so. Asserting only the final key would let a
// future change satisfy the table for the wrong reason, so both routes are checked directly.
func TestPodPayloadHasNoSubscriptionAddress(t *testing.T) {
	bare := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{UID: "3f2b0d18-0000-4000-8000-000000000001", Name: "asterisk-call-x"},
	}

	tests := []struct {
		name string
		data any
	}{
		{
			// `*corev1.Pod` is an external type and remains without a subscription-address
			// override; the checkImplements guard below pins that it never grows one by accident
			// (e.g. via a promoted method).
			"bare corev1.Pod resolves to the placeholder without implementing the interface",
			bare,
		},
		{
			// The wrapper's explicit "" is the sanctioned route to the placeholder key.
			"pod.Event wrapper's explicit empty address resolves to the placeholder",
			&pod.Event{Pod: bare},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriptionID := resolveSubscriptionID(t, tt.data)
			if subscriptionID != "" {
				t.Errorf("Wrong match. expect: empty subscription id, got: %s", subscriptionID)
			}

			if !eventtopic.IsPlaceholderSubscriptionID(subscriptionID) {
				t.Errorf("Wrong match. expect: placeholder subscription id, got: not a placeholder")
			}
		})
	}

	if _, ok := any(bare).(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("*corev1.Pod remains without a subscription-address override. it is an external type; the explicit address lives on the pod.Event wrapper.")
	}
}

// TestPodWrapperUsesDefaultSubscriptionID pins the OLD models/pod.Pod wrapper as method-free.
// The wrapper is not on any publish path, but it anonymously embeds corev1.Pod BY VALUE, so a
// method added to either would promote onto the other -- the same wrapper-promotion hazard design
// §2.4 calls out for customer.CustomerCreatedEvent. The publish-side address belongs exclusively
// to the shape-preserving pod.Event wrapper.
func TestPodWrapperUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &pod.Pod{}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("*pod.Pod remains without a subscription-address override. the explicit address lives on the pod.Event wrapper only.")
	}
}
