package aicallhandler

import (
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

	h := NewAIcallHandler(nil, nil, nil, mockCache, nil, nil, nil, nil)

	concrete, ok := h.(*aicallHandler)
	if !ok {
		t.Fatalf("NewAIcallHandler did not return an *aicallHandler")
	}
	if concrete.cache != mockCache {
		t.Errorf("cache dependency was not wired through the constructor")
	}
}
