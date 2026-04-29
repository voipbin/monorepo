package messagehandler

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// metricRegistered reports whether a metric with the given fully-qualified
// name is registered with the default Prometheus registerer.
//
// We attempt to register a throwaway counter that shares the target name. The
// default registerer rejects the registration with AlreadyRegisteredError when
// the name is already in use, which is the signal we want. Using Gather() is
// not sufficient for *Vec collectors with no observed children, since they do
// not emit a metric family until at least one label combination is touched.
func metricRegistered(name string) bool {
	probe := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: "probe",
	})
	err := prometheus.DefaultRegisterer.Register(probe)
	if err == nil {
		// Probe registered cleanly, so the real metric was NOT already there.
		// Unregister our probe so we leave the registry as we found it.
		prometheus.DefaultRegisterer.Unregister(probe)
		return false
	}
	var are prometheus.AlreadyRegisteredError
	if errors.As(err, &are) {
		return true
	}
	// CounterVec/GaugeVec collectors register a Desc whose fully-qualified
	// name plus variable label set must be unique. Re-registering a plain
	// Counter under the same name surfaces a descriptor-conflict error rather
	// than AlreadyRegisteredError. Treat any of these "name already in the
	// registry" messages as positive signal too.
	msg := err.Error()
	return strings.Contains(msg, "already registered") ||
		strings.Contains(msg, "duplicate metrics collector") ||
		strings.Contains(msg, "previously registered descriptor with the same fully-qualified name")
}

func TestPromConversationDeliveryStatusUpdateFailedTotal_registered(t *testing.T) {
	const name = "ai_manager_message_delivery_status_update_failed_total"
	if !metricRegistered(name) {
		t.Fatalf("metric %s not registered", name)
	}
}
