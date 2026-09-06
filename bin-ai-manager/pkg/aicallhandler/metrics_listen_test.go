package aicallhandler

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// Test_listenMetricNames pins the full listen metric surface and its naming.
//
// Two things it guards. (1) The set is complete -- an outcome with no metric is
// an outcome nobody can alert on, and every one of these is a real operational
// signal. (2) No Name string carries the "ai_manager_" prefix itself: the
// Prometheus client library prepends the namespace, so a literal prefix renders
// as ai_manager_ai_manager_..., which silently breaks every dashboard query
// written against the documented name.
func Test_listenMetricNames(t *testing.T) {
	// A counter with no observations may not appear in Gather() output at all,
	// so force a zero sample for each first. Done here, in the test, never in
	// production code.
	promListenStartTotal.WithLabelValues("call", "started")
	promListenSegmentTotal.WithLabelValues("buffered")
	promListenTurnTotal.WithLabelValues("call", "ran")
	promListenNotifyTotal.WithLabelValues("call")
	promListenStopFailedTotal.Add(0)
	promListenMembershipCheckFailedTotal.Add(0)
	promListenConversationSegmentTotal.WithLabelValues("buffered")
	promListenConversationFlushTotal.WithLabelValues("ran")

	expected := map[string]bool{
		"ai_manager_aicall_listen_start_total":                   false,
		"ai_manager_aicall_listen_segment_total":                 false,
		"ai_manager_aicall_listen_turn_total":                    false,
		"ai_manager_aicall_listen_notify_total":                  false,
		"ai_manager_aicall_listen_stop_failed_total":             false,
		"ai_manager_aicall_listen_membership_check_failed_total": false,
		"ai_manager_aicall_listen_conversation_segment_total":    false,
		"ai_manager_aicall_listen_conversation_flush_total":      false,
	}

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("could not gather metrics. err: %v", err)
	}

	for _, f := range families {
		name := f.GetName()
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
		if strings.HasPrefix(name, "ai_manager_ai_manager_") {
			t.Errorf("metric %q has a doubled namespace -- the Name field must not repeat the namespace prefix", name)
		}
	}

	for name, seen := range expected {
		if !seen {
			t.Errorf("metric %q is not registered", name)
		}
	}
}
