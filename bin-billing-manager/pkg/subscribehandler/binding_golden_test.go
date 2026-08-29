package subscribehandler

import (
	"testing"
)

// Test_topicPatterns_golden pins the EXACT binding patterns billing-manager binds on
// the global topic exchange `bin-manager.event` (VOIP-1406). The expected values are
// deliberate string LITERALS (never derived through eventtopic helpers) so any drift
// in either the pattern list or the eventtopic normalization fails this test. The
// list mirrors design §5 (2026-08-29-voip-1406-consumer-topic-migration-design.md)
// byte-for-byte: one pattern per dispatch pair in processEvent.
func Test_topicPatterns_golden(t *testing.T) {
	expected := []string{
		"call-manager.call.*.progressing",
		"call-manager.call.*.hangup",
		"call-manager.recording.*.started",
		"call-manager.recording.*.finished",
		"message-manager.message.*.created",
		"email-manager.email.*.created",
		"customer-manager.customer.*.deleted",
		"customer-manager.customer.*.created",
		"customer-manager.customer.*.frozen",
		"customer-manager.customer.*.recovered",
		"number-manager.number.*.created",
		"number-manager.number.*.renewed",
		"tts-manager.speaking.*.started",
		"tts-manager.speaking.*.stopped",
	}

	if len(topicPatterns) != 14 {
		t.Fatalf("Wrong match. expected: 14 patterns (design §5), got: %d. topicPatterns: %v", len(topicPatterns), topicPatterns)
	}
	if len(expected) != len(topicPatterns) {
		t.Fatalf("Wrong match. expected: %d patterns, got: %d. topicPatterns: %v", len(expected), len(topicPatterns), topicPatterns)
	}

	for i, pattern := range expected {
		if topicPatterns[i] != pattern {
			t.Errorf("Wrong match. index: %d, expected: %s, got: %s", i, pattern, topicPatterns[i])
		}
	}
}
