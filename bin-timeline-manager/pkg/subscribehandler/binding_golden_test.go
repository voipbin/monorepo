package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Test_topicPatterns_golden pins the EXACT bind set timeline-manager places on the
// global bin-manager.event topic exchange (VOIP-1406, design §5). timeline-manager is
// the archive-everything service: a single catch-all "#" binding, deliberately a
// superset of the old 25 fanout subscriptions, capturing every current and future
// topic publisher. Any change to topicPatterns must be a reviewed design decision
// that updates this golden in the same commit.
func Test_topicPatterns_golden(t *testing.T) {
	expectedPatterns := []string{
		"#",
	}

	if len(topicPatterns) != 1 {
		t.Fatalf("Wrong topicPatterns count. expect: 1, got: %d (%v)", len(topicPatterns), topicPatterns)
	}
	for i, expected := range expectedPatterns {
		if topicPatterns[i] != expected {
			t.Errorf("Wrong pattern at index %d. expect: %q, got: %q", i, expected, topicPatterns[i])
		}
	}
}

// Test_fanoutUnbindTargets_golden pins the EXACT fanout exchanges Run() unbinds after
// the topic binds succeed: all 25 non-asterisk entries of subscribeTargets, in order.
// The asterisk fanout leg is permanently retained (asterisk-proxy does not publish to
// the topic exchange) and must NEVER appear in this list.
func Test_fanoutUnbindTargets_golden(t *testing.T) {
	expectedTargets := []string{
		"bin-manager.ai-manager.event",
		"bin-manager.agent-manager.event",
		"bin-manager.billing-manager.event",
		"bin-manager.call-manager.event",
		"bin-manager.campaign-manager.event",
		"bin-manager.conference-manager.event",
		"bin-manager.contact-manager.event",
		"bin-manager.conversation-manager.event",
		"bin-manager.customer-manager.event",
		"bin-manager.email-manager.event",
		"bin-manager.flow-manager.event",
		"bin-manager.message-manager.event",
		"bin-manager.number-manager.event",
		"bin-manager.outdial-manager.event",
		"bin-manager.pipecat-manager.event",
		"bin-manager.queue-manager.event",
		"bin-manager.registrar-manager.event",
		"bin-manager.route-manager.event",
		"bin-manager.sentinel-manager.event",
		"bin-manager.storage-manager.event",
		"bin-manager.tag-manager.event",
		"bin-manager.talk-manager.event",
		"bin-manager.transcribe-manager.event",
		"bin-manager.transfer-manager.event",
		"bin-manager.tts-manager.event",
	}

	if len(fanoutUnbindTargets) != 25 {
		t.Fatalf("Wrong fanoutUnbindTargets count. expect: 25, got: %d (%v)", len(fanoutUnbindTargets), fanoutUnbindTargets)
	}
	for i, expected := range expectedTargets {
		if string(fanoutUnbindTargets[i]) != expected {
			t.Errorf("Wrong unbind target at index %d. expect: %q, got: %q", i, expected, fanoutUnbindTargets[i])
		}
	}

	for _, target := range fanoutUnbindTargets {
		if string(target) == "asterisk.all.event" {
			t.Errorf("fanoutUnbindTargets must NEVER contain the retained asterisk fanout leg, got: %v", fanoutUnbindTargets)
		}
	}
}

// Test_retainedFanoutTargets_golden pins the legs that stay OUTSIDE the topic
// migration: the asterisk fanout subscription remains in subscribeTargets (retained
// permanently -- asterisk-proxy does not publish to the topic exchange), and the
// VOIP-1258 "#" bind on the webhook-manager topic exchange (a separate, permanently
// coexisting exchange) remains in Run().
func Test_retainedFanoutTargets_golden(t *testing.T) {
	foundAsterisk := false
	for _, target := range subscribeTargets {
		if string(target) == "asterisk.all.event" {
			foundAsterisk = true
		}
	}
	if !foundAsterisk {
		t.Errorf("subscribeTargets must retain the asterisk fanout leg %q, got: %v", "asterisk.all.event", subscribeTargets)
	}

	if len(subscribeTargets) != 26 {
		t.Errorf("Wrong subscribeTargets count. expect: 26, got: %d", len(subscribeTargets))
	}
	if len(subscribeTargets)-len(fanoutUnbindTargets) != 1 {
		t.Errorf("fanoutUnbindTargets must be subscribeTargets minus exactly the asterisk leg. subscribeTargets: %d, fanoutUnbindTargets: %d", len(subscribeTargets), len(fanoutUnbindTargets))
	}

	// The retained VOIP-1258 webhook topic exchange bind: pin the exchange name so a
	// rename or accidental removal of the constant surfaces here.
	if string(commonoutline.QueueNameWebhookEventTopic) != "bin-manager.webhook-manager.event.topic" {
		t.Errorf("Wrong webhook topic exchange name. expect: %q, got: %q", "bin-manager.webhook-manager.event.topic", commonoutline.QueueNameWebhookEventTopic)
	}

	// And the global topic exchange the new "#" bind targets.
	if string(commonoutline.QueueNameEvent) != "bin-manager.event" {
		t.Errorf("Wrong global topic exchange name. expect: %q, got: %q", "bin-manager.event", commonoutline.QueueNameEvent)
	}
}
