// Package container holds the publish-side event model of sentinel-manager's Docker backend
// (VOIP-1418).
//
// It replaces the former `models/pod` package, whose payload was a raw Kubernetes `*corev1.Pod`.
// Docker has no equivalent object, and the recovery contract only ever needed three fields, so
// this model is a deliberate minimum rather than a translation of the Kubernetes shape.
package container

// list of watched service names.
//
// The value is the logical Asterisk workload behind a watched container, derived from the
// container's own name prefix (see pkg/dockerwatchhandler). It is the field consumers filter on
// -- bin-call-manager's recovery path matches ServiceAsteriskCall.
const (
	ServiceAsteriskCall       = "asterisk-call"
	ServiceAsteriskConference = "asterisk-conference"
	ServiceAsteriskRegistrar  = "asterisk-registrar"
)

// list of event types.
//
// These replace the former `pod_updated`/`pod_deleted` pair. The wire strings are renamed, not
// reused: "pod" is actively misleading once nothing is a pod, and sentinel-manager's only real
// consumer (bin-call-manager) is updated in the same change, so there is no external subscriber
// this rename can silently break (design §3.5).
//
// `container_started` is the analogue of the old `pod_updated`: Docker has no watch-resync
// distinction, so `start` is the natural "this container changed state" signal.
const (
	EventTypeContainerStarted = "container_started"
	EventTypeContainerDied    = "container_died"
)

// Event is the payload sentinel-manager publishes for a watched container's lifecycle
// transition (design §3.5).
//
// AsteriskID is resolved BEFORE the death, by the state table in pkg/dockerwatchhandler, never
// reactively at die time -- a dying container's inspect response has an empty IPAddress, and a
// die-time Redis scan cannot distinguish "the id that just died" from "the id that just took
// over the same IP". It is the empty string when the container died before its id could ever be
// resolved (design §3.3 step 3); consumers MUST guard on that rather than passing it downstream.
type Event struct {
	ContainerName string `json:"container_name"`
	Service       string `json:"service"`
	AsteriskID    string `json:"asterisk_id"`
}

// EventSubscriptionID returns the subscription address of this type on the global topic exchange
// `bin-manager.event` (VOIP-1404 §4.2, VOIP-1419).
//
// This is a deliberate departure from the old `pod.Event`, which returned "" unconditionally
// because a Kubernetes Pod carried no addressable top-level identity at all. A resolved
// asterisk-id IS a real, stable address for the container generation the event describes, and
// treating it as a permanent placeholder would waste the identity the state table exists to
// resolve (design §6).
//
// An unresolved (empty) id degrades to the `-` placeholder through the standard
// eventtopic.normalizeSubscriptionID path -- no special casing here. bin-call-manager's
// subscription binds the wildcard pattern `sentinel-manager.container.*.died`, so populating a
// real address is additive: it opens instance subscription to a future consumer without changing
// anything for today's only consumer.
//
// The receiver is a POINTER per the eventtopic.SubscriptionIdentifier contract; a nil receiver
// is guarded because the publish path resolves the address by interface assertion, and a typed
// nil satisfies the assertion.
func (h *Event) EventSubscriptionID() string {
	if h == nil {
		return ""
	}

	return h.AsteriskID
}
