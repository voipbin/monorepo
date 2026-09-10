package aicallhandler

import (
	"strings"
	"testing"

	"monorepo/bin-ai-manager/pkg/cachehandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_NewAIcallHandler_WiresCache pins that the cache dependency reaches the
// struct. Every listen path (the transcribe resolver set, the transcript
// buffers, the debounce lock, the turn counter, and ToolHandle's listen-turn
// membership check) goes through it; a constructor that silently dropped it
// would nil-panic at the first transcript segment, in production, on a code
// path no unit test with an explicitly-constructed handler would ever reach.
func Test_NewAIcallHandler_WiresCache(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := NewAIcallHandler(nil, nil, nil, mockCache, nil, nil, nil, nil, nil, nil, nil)

	concrete, ok := h.(*aicallHandler)
	if !ok {
		t.Fatalf("NewAIcallHandler did not return an *aicallHandler")
	}
	if concrete.cache != mockCache {
		t.Errorf("cache dependency was not wired through the constructor")
	}
}

// Test_InsightSystemPrompt_RunLLMGuidance is a deliberate change-detector for
// the Insight prompt wording added by VOIP-1485. The prompt is the only place
// that tells the model what run_llm costs across the whole tool set, so an
// edit that drops the guidance must break a test rather than ship silently.
//
// The listen-turn prompts are asserted clean on purpose: InsightSystemPrompt is
// also message #1 of every listen turn, where the model is talking to nobody,
// so the run_llm carve-out is phrased conditionally and lives in the shared
// Insight prompt only. Repeating it in the listen prompts would contradict
// their "you are not talking to anyone" framing.
func Test_InsightSystemPrompt_RunLLMGuidance(t *testing.T) {
	wants := []string{
		"run_llm parameter",
		"notify_agent takes no run_llm argument",
	}
	for _, want := range wants {
		if !strings.Contains(InsightSystemPrompt, want) {
			t.Errorf("InsightSystemPrompt must contain %q.\ngot: %s", want, InsightSystemPrompt)
		}
	}

	listenPrompts := map[string]string{
		"ListenTurnSystemPrompt":             ListenTurnSystemPrompt,
		"ListenTurnConversationSystemPrompt": ListenTurnConversationSystemPrompt,
	}
	for name, prompt := range listenPrompts {
		if strings.Contains(prompt, "run_llm") {
			t.Errorf("%s must not mention run_llm; the guidance belongs to InsightSystemPrompt only.\ngot: %s", name, prompt)
		}
	}
}
