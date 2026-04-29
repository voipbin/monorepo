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
