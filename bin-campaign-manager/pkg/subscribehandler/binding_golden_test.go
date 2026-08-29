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
