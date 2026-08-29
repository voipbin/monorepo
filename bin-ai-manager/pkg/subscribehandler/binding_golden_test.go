package subscribehandler

import (
	"testing"
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
	}

	// design §5: ai-manager binds exactly 10 patterns.
	if len(topicPatterns) != 10 {
		t.Fatalf("topicPatterns count mismatch. expected: 10, got: %d (%v)", len(topicPatterns), topicPatterns)
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

// Test_topicPatterns_excludesConferenceUpdated is the §4 negative assertion: the
// `conference-manager/conference_updated` dispatch case is unreachable today and MUST
// stay unreachable -- VOIP-1406 changes where events come from, never what is processed.
// Binding this pattern would activate the dead case (an unreviewed behavior change).
// Follow-up VOIP-1422 decides whether to activate or delete the case.
func Test_topicPatterns_excludesConferenceUpdated(t *testing.T) {
	excluded := "conference-manager.conference.*.updated"

	for _, pattern := range topicPatterns {
		if pattern == excluded {
			t.Errorf("topicPatterns must NOT contain the excluded pattern %q (VOIP-1406 design §4, follow-up VOIP-1422)", excluded)
		}
	}
}
