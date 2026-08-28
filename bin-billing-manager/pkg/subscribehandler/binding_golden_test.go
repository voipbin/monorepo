package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
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

// Test_fanoutUnbindTargets_golden pins the fanout event exchanges billing-manager
// unbinds after a fully successful topic-bind pass (VOIP-1406). This must equal the
// cmd wiring's subscribeTargets: billing-manager has no asterisk leg, so ALL of its
// fanout subscriptions are unbind targets.
func Test_fanoutUnbindTargets_golden(t *testing.T) {
	expected := []string{
		"bin-manager.call-manager.event",
		"bin-manager.message-manager.event",
		"bin-manager.email-manager.event",
		"bin-manager.customer-manager.event",
		"bin-manager.number-manager.event",
		"bin-manager.tts-manager.event",
	}

	if len(fanoutUnbindTargets) != len(expected) {
		t.Fatalf("Wrong match. expected: %d targets, got: %d. fanoutUnbindTargets: %v", len(expected), len(fanoutUnbindTargets), fanoutUnbindTargets)
	}

	for i, target := range expected {
		if fanoutUnbindTargets[i] != target {
			t.Errorf("Wrong match. index: %d, expected: %s, got: %s", i, target, fanoutUnbindTargets[i])
		}
	}

	// The asterisk fanout exchange belongs only to call-manager/timeline-manager;
	// billing-manager must never unbind (nor bind) it.
	for _, target := range fanoutUnbindTargets {
		if target == string(commonoutline.QueueNameAsteriskEventAll) {
			t.Errorf("Wrong match. fanoutUnbindTargets must not contain the asterisk exchange. got: %v", fanoutUnbindTargets)
		}
	}
}
