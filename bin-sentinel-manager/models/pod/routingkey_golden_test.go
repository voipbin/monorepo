// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type sentinel-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site.
//
// sentinel-manager is the ONE documented placeholder-by-design publisher (design §2.4 / §5).
// pkg/monitoringhandler/run.go publishes the raw Kubernetes `*corev1.Pod` it received from the
// informer, not a VoIPbin model. A `corev1.Pod` has no top-level `id` in its JSON form -- its
// identity lives under `metadata` -- so the default resolution yields "" and
// eventtopic.RoutingKey collapses the subscription-address segment to the `-` placeholder. The
// keys are therefore `sentinel-manager.pod.-.updated` / `.deleted` for EVERY pod, and instance
// subscription of pod events is not supported. The runbook consequence is recorded in
// docs/architecture.md: `sentinel_manager_topic_placeholder_total ~= topic_publish_total{ok}` is
// the HEALTHY invariant here, not an alert condition -- but the ratio still detects publish
// regressions.
//
// Do NOT "fix" this by attaching an override to `*corev1.Pod` (impossible -- external type) or by
// wrapping the payload in models/pod.Pod at the publish site (that would change the payload shape
// for every existing fanout consumer). The placeholder is the accepted trade-off.
//
// MAINTENANCE: `pod_added` (EventTypePodAdded) is a DEAD constant -- the informer's AddFunc is an
// intentional no-op (Kubernetes replays existing pods as Add during the initial list), so nothing
// publishes it. It is deliberately absent from the table; if AddFunc ever starts publishing, add
// its row in the same change.
package pod_test

import (
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-sentinel-manager/models/pod"
)

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
	publisher := string(commonoutline.ServiceNameSentinelManager)

	// The exact value monitoringhandler hands to PublishEvent: the informer's own object.
	// Populated with a realistic uid/name to make the point that NONE of it reaches the key.
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
			podData,
			"sentinel-manager.pod.-.updated",
		},
		// pkg/monitoringhandler/run.go:117 -- informer DeleteFunc.
		{
			"pod_deleted (placeholder address by design)",
			pod.EventTypePodDeleted,
			podData,
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

// TestPodPayloadHasNoSubscriptionAddress pins the two independent facts that together produce the
// placeholder: the published type carries no override, AND its marshaled form has no top-level
// `id`. Asserting only the final key would let a future change satisfy the table for the wrong
// reason (e.g. an override returning ""), so both halves are checked directly.
func TestPodPayloadHasNoSubscriptionAddress(t *testing.T) {
	var data any = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{UID: "3f2b0d18-0000-4000-8000-000000000001", Name: "asterisk-call-x"},
	}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("*corev1.Pod must not implement SubscriptionIdentifier. it is an external type and the placeholder address is deliberate.")
	}

	subscriptionID := resolveSubscriptionID(t, data)
	if subscriptionID != "" {
		t.Errorf("Wrong match. expect: empty subscription id, got: %s", subscriptionID)
	}

	if !eventtopic.IsPlaceholderSubscriptionID(subscriptionID) {
		t.Errorf("Wrong match. expect: placeholder subscription id, got: not a placeholder")
	}
}

// TestPodWrapperUsesDefaultSubscriptionID pins the models/pod.Pod wrapper as override-free too.
// The wrapper is not on any publish path today, but it anonymously embeds corev1.Pod, so a method
// added to either would promote onto it -- the same wrapper-promotion hazard design §2.4 calls out
// for customer.CustomerCreatedEvent.
func TestPodWrapperUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &pod.Pod{}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("*pod.Pod must not implement SubscriptionIdentifier. sentinel-manager has no subscription-id override.")
	}
}
