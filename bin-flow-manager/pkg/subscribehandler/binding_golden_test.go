package subscribehandler

import (
	"testing"
)

// Test_topicPatterns_golden pins the EXACT bind set of flow-manager on the global topic
// exchange `bin-manager.event` (VOIP-1406 design §5). The expected values are literal
// strings on purpose: the production list is built via eventtopic.PatternAction, so this
// test catches drift in either direction -- a dispatch case added without a pattern, or a
// pattern added without a dispatch case -- as well as any normalization change inside
// eventtopic.
func Test_topicPatterns_golden(t *testing.T) {
	expected := []string{
		"customer-manager.customer.*.deleted",
	}

	// design §5: flow-manager binds exactly 1 pattern.
	if len(topicPatterns) != 1 {
		t.Fatalf("topicPatterns count mismatch. expected: 1, got: %d (%v)", len(topicPatterns), topicPatterns)
	}
	if len(topicPatterns) != len(expected) {
		t.Fatalf("topicPatterns count mismatch. expected: %d, got: %d (%v)", len(expected), len(topicPatterns), topicPatterns)
	}

	for i, pattern := range expected {
		if topicPatterns[i] != pattern {
			t.Errorf("topicPatterns[%d] mismatch. expected: %q, got: %q", i, pattern, topicPatterns[i])
		}
	}
}

// Test_topicPatterns_excludesCallHangup is the §4 negative assertion: the
// `call-manager/call_hangup` dispatch case is unreachable today (subscribeTargets is
// customer-only, so EventCallHangup has NEVER run in production) and MUST stay
// unreachable -- VOIP-1406 changes where events come from, never what is processed.
// Binding this pattern would activate the dead case (an unreviewed behavior change;
// activeflow cleanup on hangup is a latent-bug candidate). Follow-up VOIP-1422 decides
// whether to activate or delete the case.
func Test_topicPatterns_excludesCallHangup(t *testing.T) {
	excluded := "call-manager.call.*.hangup"

	for _, pattern := range topicPatterns {
		if pattern == excluded {
			t.Errorf("topicPatterns must NOT contain the excluded pattern %q (VOIP-1406 design §4, follow-up VOIP-1422)", excluded)
		}
	}
}

// Test_fanoutUnbindTargets_golden pins the fanout exchanges flow-manager unbinds after a
// fully successful topic bind (VOIP-1406). This is the FULL subscribeTargets set wired in
// cmd/flow-manager: the customer-manager event exchange only.
func Test_fanoutUnbindTargets_golden(t *testing.T) {
	expected := []string{
		"bin-manager.customer-manager.event",
	}

	if len(fanoutUnbindTargets) != len(expected) {
		t.Fatalf("fanoutUnbindTargets count mismatch. expected: %d, got: %d (%v)", len(expected), len(fanoutUnbindTargets), fanoutUnbindTargets)
	}

	for i, target := range expected {
		if fanoutUnbindTargets[i] != target {
			t.Errorf("fanoutUnbindTargets[%d] mismatch. expected: %q, got: %q", i, target, fanoutUnbindTargets[i])
		}
	}
}
