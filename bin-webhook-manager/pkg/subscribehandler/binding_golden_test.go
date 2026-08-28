package subscribehandler

import (
	"testing"
)

// Test_BindingGolden_TopicPatterns pins the EXACT set of bin-manager.event binding patterns
// webhook-manager uses (VOIP-1406 design §5, normative). The expected values are deliberate
// string literals, NOT eventtopic calls: the test must fail if the pattern builder or the
// production list drifts in either direction (a dispatch case added without a pattern, or a
// pattern added without a case).
func Test_BindingGolden_TopicPatterns(t *testing.T) {
	expected := []string{
		"customer-manager.customer.*.created",
		"customer-manager.customer.*.updated",
		"flow-manager.activeflow.*.created",
		"flow-manager.activeflow.*.updated",
		"flow-manager.activeflow.*.deleted",
	}

	if len(topicPatterns) != 5 {
		t.Fatalf("Wrong topicPatterns count. expect: 5, got: %d (%v)", len(topicPatterns), topicPatterns)
	}

	for i, pattern := range expected {
		if topicPatterns[i] != pattern {
			t.Errorf("Wrong topicPatterns[%d]. expect: %s, got: %s", i, pattern, topicPatterns[i])
		}
	}
}

// Test_BindingGolden_FanoutUnbindTargets pins the legacy fanout exchanges webhook-manager
// unbinds after a full topic-bind success. webhook-manager has no retained fanout leg
// (no asterisk subscription), so this is the complete subscribe-target set that cmd joins
// into the comma-joined subscribeTargets string.
func Test_BindingGolden_FanoutUnbindTargets(t *testing.T) {
	expected := []string{
		"bin-manager.customer-manager.event",
		"bin-manager.flow-manager.event",
	}

	if len(fanoutUnbindTargets) != len(expected) {
		t.Fatalf("Wrong fanoutUnbindTargets count. expect: %d, got: %d (%v)", len(expected), len(fanoutUnbindTargets), fanoutUnbindTargets)
	}

	for i, target := range expected {
		if fanoutUnbindTargets[i] != target {
			t.Errorf("Wrong fanoutUnbindTargets[%d]. expect: %s, got: %s", i, target, fanoutUnbindTargets[i])
		}
	}
}
