package subscribehandler

import (
	"testing"
)

// Test_topicPatterns_Golden pins the EXACT bind set of bin-campaign-manager on the
// global topic exchange `bin-manager.event` (VOIP-1406, design §5). The expected
// values are deliberate string literals, NOT eventtopic calls: the test must fail
// if either the production list or the eventtopic normalization drifts.
//
// Any change here means the service's event intake contract changed: keep this
// list, the dispatch switch in processEvent, and docs/architecture.md in sync.
func Test_topicPatterns_Golden(t *testing.T) {
	expectedPatterns := []string{
		"call-manager.call.*.hangup",
		"flow-manager.activeflow.*.deleted",
	}

	if len(topicPatterns) != 2 {
		t.Fatalf("topicPatterns count mismatch. expected: 2, got: %d (%v)", len(topicPatterns), topicPatterns)
	}

	for i, expected := range expectedPatterns {
		if topicPatterns[i] != expected {
			t.Errorf("topicPatterns[%d] mismatch. expected: %s, got: %s", i, expected, topicPatterns[i])
		}
	}
}

// Test_fanoutUnbindTargets_Golden pins the fanout event exchanges the subscribe
// queue unbinds from after a fully successful topic binding (VOIP-1406). It must
// equal the service's fanout subscribeTargets (campaign-manager has no retained
// asterisk leg).
func Test_fanoutUnbindTargets_Golden(t *testing.T) {
	expectedTargets := []string{
		"bin-manager.call-manager.event",
		"bin-manager.flow-manager.event",
	}

	if len(fanoutUnbindTargets) != len(expectedTargets) {
		t.Fatalf("fanoutUnbindTargets count mismatch. expected: %d, got: %d (%v)", len(expectedTargets), len(fanoutUnbindTargets), fanoutUnbindTargets)
	}

	for i, expected := range expectedTargets {
		if fanoutUnbindTargets[i] != expected {
			t.Errorf("fanoutUnbindTargets[%d] mismatch. expected: %s, got: %s", i, expected, fanoutUnbindTargets[i])
		}
	}
}
