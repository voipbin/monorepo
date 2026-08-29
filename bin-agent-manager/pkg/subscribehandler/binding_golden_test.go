package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Test_topicPatterns_Golden pins the EXACT bind set of bin-agent-manager on the
// global topic exchange `bin-manager.event` (VOIP-1406, design §5). The expected
// values are deliberate string literals, NOT eventtopic calls: the test must fail
// if either the production list or the eventtopic normalization drifts.
//
// Any change here means the service's event intake contract changed: keep this
// list, the dispatch switch in processEvent, and docs/architecture.md in sync.
func Test_topicPatterns_Golden(t *testing.T) {
	expectedPatterns := []string{
		"call-manager.groupcall.*.created",
		"call-manager.groupcall.*.progressing",
		"customer-manager.customer.*.deleted",
		"customer-manager.customer.*.created",
	}

	if len(topicPatterns) != 4 {
		t.Fatalf("topicPatterns count mismatch. expected: 4, got: %d (%v)", len(topicPatterns), topicPatterns)
	}

	for i, expected := range expectedPatterns {
		if topicPatterns[i] != expected {
			t.Errorf("topicPatterns[%d] mismatch. expected: %s, got: %s", i, expected, topicPatterns[i])
		}
	}
}

// Test_fanoutUnbindTargets_Retains1258TopicBind pins that the VOIP-1258
// webhook-topic cutover is untouched by VOIP-1407/VOIP-1406: the 1258 topic
// exchange the queue binds with "#" inside Run() is a different exchange than the
// global topic exchange bin-manager.event, and its "#" pattern is NOT in
// topicPatterns (the 1258 bind is a separate, retained bind on a different
// exchange, unrelated to the fanout-vs-topic cutover VOIP-1407 performs).
func Test_fanoutUnbindTargets_Retains1258TopicBind(t *testing.T) {
	if string(commonoutline.QueueNameWebhookEventTopic) != "bin-manager.webhook-manager.event.topic" {
		t.Errorf("QueueNameWebhookEventTopic value drifted. expected: bin-manager.webhook-manager.event.topic, got: %s", string(commonoutline.QueueNameWebhookEventTopic))
	}

	// the VOIP-1258 topic exchange must stay distinct from the VOIP-1406 global
	// topic exchange bin-manager.event -- the retained block binds/unbinds on its
	// own exchange, never on bin-manager.event.
	if string(commonoutline.QueueNameWebhookEventTopic) == string(commonoutline.QueueNameEvent) {
		t.Errorf("QueueNameWebhookEventTopic must NOT equal the global topic exchange QueueNameEvent (%s) -- the retained VOIP-1258 block binds on a separate exchange", string(commonoutline.QueueNameEvent))
	}

	for _, pattern := range topicPatterns {
		if pattern == "#" {
			t.Errorf("topicPatterns must NOT contain the 1258 catch-all pattern \"#\" -- the 1258 bind lives on its own exchange, not bin-manager.event")
		}
	}
}
