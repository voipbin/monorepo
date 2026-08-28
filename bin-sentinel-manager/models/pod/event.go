package pod

import corev1 "k8s.io/api/core/v1"

const (
	EventTypePodAdded   = "pod_added"
	EventTypePodUpdated = "pod_updated"
	EventTypePodDeleted = "pod_deleted"
)

// Event is the publish-side wrapper for pod lifecycle events (VOIP-1419). `*corev1.Pod` is an
// external type and cannot carry EventSubscriptionID itself, so the publish sites
// (pkg/monitoringhandler/run.go) wrap the informer's pod in this type. The anonymous POINTER
// embed marshals byte-identically to the bare `*corev1.Pod` (the embed inlines and Pod has no
// MarshalJSON), so fanout consumers see the exact same payload; a nil embed marshals to `{}`.
type Event struct {
	*corev1.Pod
}

// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event` (VOIP-1404 §4.2, VOIP-1419). Pod events carry no top-level id
// (a Pod's identity lives under `metadata`), so this returns "" — placeholder-by-design,
// preserving the invariant `placeholder_total ≈ publish_total{ok}`.
func (h *Event) EventSubscriptionID() string {
	return ""
}
