package subscribehandler

import (
	"testing"
)

// Test_topicPatterns_golden pins the EXACT bind set of queue-manager on the global topic
// exchange `bin-manager.event` (VOIP-1406 design §5). The expected values are literal
// strings on purpose: the production list is built via eventtopic.PatternAction, so this
// test catches drift in either direction -- a dispatch case added without a pattern, or a
// pattern added without a dispatch case -- as well as any normalization change inside
// eventtopic.
func Test_topicPatterns_golden(t *testing.T) {
	expected := []string{
		"call-manager.call.*.hangup",
		"call-manager.confbridge.*.joined",
		"call-manager.confbridge.*.leaved",
	}

	// design §5: queue-manager binds exactly 3 patterns.
	if len(topicPatterns) != 3 {
		t.Fatalf("topicPatterns count mismatch. expected: 3, got: %d (%v)", len(topicPatterns), topicPatterns)
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

// Test_topicPatterns_excludesCustomerDeleted is the §4 negative assertion: the
// `customer-manager/customer_deleted` dispatch case is unreachable today (the customer
// fanout exchange was never subscribed) and MUST stay unreachable -- VOIP-1406 changes
// where events come from, never what is processed. Binding this pattern would activate
// the dead case (an unreviewed behavior change; likely-latent queue cleanup on customer
// deletion). Follow-up VOIP-1422 decides whether to activate or delete the case.
func Test_topicPatterns_excludesCustomerDeleted(t *testing.T) {
	excluded := "customer-manager.customer.*.deleted"

	for _, pattern := range topicPatterns {
		if pattern == excluded {
			t.Errorf("topicPatterns must NOT contain the excluded pattern %q (VOIP-1406 design §4, follow-up VOIP-1422)", excluded)
		}
	}
}
