package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
)

// Test_topicPatterns_golden pins the EXACT bind set of ai-manager on the global topic
// exchange `bin-manager.event` (VOIP-1406 design §5). The expected values are literal
// strings on purpose: the production list is built via eventtopic.PatternAction, so this
// test catches drift in either direction -- a dispatch case added without a pattern, or a
// pattern added without a dispatch case -- as well as any normalization change inside
// eventtopic.
func Test_topicPatterns_golden(t *testing.T) {
	expected := []string{
		"call-manager.confbridge.*.joined",
		"call-manager.confbridge.*.leaved",
		"call-manager.call.*.hangup",
		"call-manager.dtmf.*.received",
		"pipecat-manager.message.*.user_transcription",
		"pipecat-manager.message.*.bot_llm",
		"pipecat-manager.message.*.bot_llm_intermediate",
		"pipecat-manager.pipecatcall.*.initialized",
		"pipecat-manager.pipecatcall.*.terminated",
		"pipecat-manager.team.*.member_switched",
		"conference-manager.conference.*.deleted",
		"transcribe-manager.transcript.*.created",
		"conversation-manager.conversation.*.message_created",
	}

	// design §5 + VOIP-1422 + NOJIRA Insight AI realtime listen + VOIP-1470:
	// ai-manager binds exactly 13 patterns.
	if len(topicPatterns) != 13 {
		t.Fatalf("topicPatterns count mismatch. expected: 13, got: %d (%v)", len(topicPatterns), topicPatterns)
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

// Test_conversationBindingMatchesProducer pins that the bound pattern is what
// conversation-manager actually publishes for a message, so a renamed event
// type fails here rather than in production.
func Test_conversationBindingMatchesProducer(t *testing.T) {
	want := eventtopic.PatternForEventType(string(commonoutline.ServiceNameConversationManager), cvmessage.EventTypeMessageCreated)
	if want != "conversation-manager.conversation.*.message_created" {
		t.Fatalf("pattern mismatch. got: %q", want)
	}
	msg := &cvmessage.Message{ConversationID: uuid.FromStringOrNil("66660000-0000-4000-8000-000000000001")}
	key := eventtopic.RoutingKey(string(commonoutline.ServiceNameConversationManager), cvmessage.EventTypeMessageCreated, msg.EventSubscriptionID())
	if key != "conversation-manager.conversation.66660000-0000-4000-8000-000000000001.message_created" {
		t.Errorf("routing key mismatch. got: %q", key)
	}
}
