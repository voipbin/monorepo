package pipecatcall

import (
	"testing"

	"github.com/gofrs/uuid"
)

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

// TestSession_PendingInReplyToMessageID_concurrentAccess is a race-detector
// regression test (VOIP-1234 §4-1). SetPendingInReplyToMessageID is called
// from the RabbitMQ RPC worker pool (SendMessage); SnapshotPendingInReplyToMessageID
// is called from the WebSocket read loop goroutine. Direct field access here
// would trip `go test -race` because uuid.UUID is a plain [16]byte array with
// no atomicity guarantee. Run with `go test -race` to verify.
func TestSession_PendingInReplyToMessageID_concurrentAccess(t *testing.T) {
	s := &Session{}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			id := uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
			s.SetPendingInReplyToMessageID(id)
		}
	}()

	for i := 0; i < 1000; i++ {
		_ = s.SnapshotPendingInReplyToMessageID()
	}
	<-done
}
