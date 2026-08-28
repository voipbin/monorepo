package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Test_topicPatterns_golden pins the EXACT pattern strings this service binds on the
// global topic exchange `bin-manager.event` (VOIP-1406, design §5: call-manager).
// The expected values are deliberate string literals, NOT eventtopic calls: this is the
// consumer-side sibling of the routing-key golden tests, and it must catch drift in
// either direction -- a dispatch case added without a pattern, or a pattern added
// without a case, or a change in the eventtopic normalization itself.
func Test_topicPatterns_golden(t *testing.T) {
	expectedPatterns := []string{
		"customer-manager.customer.*.deleted",
		"customer-manager.customer.*.frozen",
		"flow-manager.activeflow.*.updated",
		"sentinel-manager.pod.*.deleted",
	}

	if len(topicPatterns) != 4 {
		t.Errorf("Wrong pattern count. expect: 4, got: %d (%v)", len(topicPatterns), topicPatterns)
	}

	if len(topicPatterns) != len(expectedPatterns) {
		t.Fatalf("Wrong match. expect: %v, got: %v", expectedPatterns, topicPatterns)
	}
	for i, expected := range expectedPatterns {
		if topicPatterns[i] != expected {
			t.Errorf("Wrong pattern at index %d. expect: %s, got: %s", i, expected, topicPatterns[i])
		}
	}
}

// Test_fanoutUnbindTargets_golden pins the fanout event exchanges unbound after a full
// topic-bind success, and pins the RETAINED asterisk fanout target: asterisk-proxy does
// not publish to the topic exchange, so `asterisk.all.event` must NEVER appear in the
// unbind set.
func Test_fanoutUnbindTargets_golden(t *testing.T) {
	expectedTargets := []string{
		"bin-manager.customer-manager.event",
		"bin-manager.flow-manager.event",
		"bin-manager.sentinel-manager.event",
	}

	if len(fanoutUnbindTargets) != len(expectedTargets) {
		t.Fatalf("Wrong match. expect: %v, got: %v", expectedTargets, fanoutUnbindTargets)
	}
	for i, expected := range expectedTargets {
		if fanoutUnbindTargets[i] != expected {
			t.Errorf("Wrong target at index %d. expect: %s, got: %s", i, expected, fanoutUnbindTargets[i])
		}
	}

	// the retained asterisk leg: pin the constant's literal value and assert it is
	// excluded from the unbind set.
	retained := "asterisk.all.event"
	if string(commonoutline.QueueNameAsteriskEventAll) != retained {
		t.Errorf("Wrong retained asterisk target. expect: %s, got: %s", retained, string(commonoutline.QueueNameAsteriskEventAll))
	}
	for _, target := range fanoutUnbindTargets {
		if target == retained {
			t.Errorf("The retained asterisk fanout target must NOT be unbound. target: %s", target)
		}
	}
}
