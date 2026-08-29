package subscribehandler

import (
	"testing"
)

// Test_topicPatterns_golden pins the EXACT bind set of contact-manager on the global
// topic exchange `bin-manager.event` (VOIP-1406 design §5). The expected values are
// literal strings on purpose: the production list is built via eventtopic.PatternAction,
// so this test catches drift in either direction -- a dispatch case added without a
// pattern, or a pattern added without a dispatch case -- as well as any normalization
// change inside eventtopic.
func Test_topicPatterns_golden(t *testing.T) {
	expected := []string{
		"customer-manager.customer.*.deleted",
	}

	// design §5: contact-manager binds exactly 1 pattern.
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
