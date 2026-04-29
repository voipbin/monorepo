package pipecatcall

import "testing"

func TestSession_FlushPrimitives(t *testing.T) {
	s := &Session{}
	// Channels are zero-valued; Once and atomic.Int32 are usable in zero-value.
	s.LLMFlushOnce.Do(func() {}) // should not panic
	if got := s.LLMStopReason.Load(); got != 0 {
		t.Fatalf("expected initial reason 0, got %d", got)
	}
}

func TestSession_LLMFlushing_isAtomicBool(t *testing.T) {
	s := &Session{}
	s.LLMFlushing.Store(true)
	if !s.LLMFlushing.Load() {
		t.Fatalf("expected LLMFlushing true after Store(true)")
	}
	s.LLMFlushing.Store(false)
	if s.LLMFlushing.Load() {
		t.Fatalf("expected LLMFlushing false after Store(false)")
	}
}
