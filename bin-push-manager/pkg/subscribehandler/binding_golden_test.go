package subscribehandler

import "testing"

// Test_topicPatterns_golden pins the exact bind set of push-manager on the global topic exchange. The
// expected values are literal strings on purpose, so drift in either direction (a dispatch case without a
// pattern, or a pattern without a dispatch case) is caught.
func Test_topicPatterns_golden(t *testing.T) {
	expected := []string{
		"agent-manager.agent.*.deleted",
		"customer-manager.customer.*.deleted",
	}

	if len(topicPatterns) != len(expected) {
		t.Fatalf("Wrong match. expect: %v, got: %v", expected, topicPatterns)
	}
	for i, pattern := range expected {
		if topicPatterns[i] != pattern {
			t.Errorf("Wrong match. index: %d, expect: %q, got: %q", i, pattern, topicPatterns[i])
		}
	}
}
