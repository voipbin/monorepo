package circuitbreakerhandler

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestBreakerStartsClosed(t *testing.T) {
	b := newBreaker()
	if b.getState() != StateClosed {
		t.Errorf("expected StateClosed, got %v", b.getState())
	}
}

func TestBreakerAllowsRequestsWhenClosed(t *testing.T) {
	b := newBreaker()
	if err := b.allow(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestBreakerTransitionsToOpenAfterThresholdFailures(t *testing.T) {
	b := newBreaker()
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}
	if b.getState() != StateOpen {
		t.Errorf("expected StateOpen, got %v", b.getState())
	}
}

func TestBreakerRejectsWhenOpen(t *testing.T) {
	b := newBreaker()
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}
	err := b.allow()
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreakerTransitionsToHalfOpenAfterTimeout(t *testing.T) {
	now := time.Now()
	b := newBreaker()
	b.nowFunc = func() time.Time { return now }

	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}

	now = now.Add(defaultOpenDuration + time.Second)
	// allow() transitions Open -> HalfOpen and lets the probe through
	if err := b.allow(); err != nil {
		t.Errorf("expected probe allowed after timeout, got %v", err)
	}
	if b.getState() != StateHalfOpen {
		t.Errorf("expected StateHalfOpen after probe, got %v", b.getState())
	}
}

func TestBreakerHalfOpenProbeSuccessCloses(t *testing.T) {
	now := time.Now()
	b := newBreaker()
	b.nowFunc = func() time.Time { return now }

	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}

	now = now.Add(defaultOpenDuration + time.Second)

	if err := b.allow(); err != nil {
		t.Errorf("expected probe to be allowed, got %v", err)
	}

	b.recordSuccess()
	if b.getState() != StateClosed {
		t.Errorf("expected StateClosed after successful probe, got %v", b.getState())
	}
}

func TestBreakerHalfOpenProbeFailureReopens(t *testing.T) {
	now := time.Now()
	b := newBreaker()
	b.nowFunc = func() time.Time { return now }

	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}

	now = now.Add(defaultOpenDuration + time.Second)
	_ = b.allow()
	b.recordFailure()

	if b.getState() != StateOpen {
		t.Errorf("expected StateOpen after failed probe, got %v", b.getState())
	}
}

func TestBreakerSuccessResetsFailureCount(t *testing.T) {
	b := newBreaker()

	for i := 0; i < defaultFailureThreshold-1; i++ {
		_ = b.allow()
		b.recordFailure()
	}

	b.recordSuccess()

	_ = b.allow()
	b.recordFailure()

	if b.getState() != StateClosed {
		t.Errorf("expected StateClosed after reset, got %v", b.getState())
	}
}

func TestBreakerDoesNotTripBelowThreshold(t *testing.T) {
	b := newBreaker()
	for i := 0; i < defaultFailureThreshold-1; i++ {
		_ = b.allow()
		b.recordFailure()
	}
	if b.getState() != StateClosed {
		t.Errorf("expected StateClosed with %d failures (threshold %d), got %v",
			defaultFailureThreshold-1, defaultFailureThreshold, b.getState())
	}
}

func TestBreakerHalfOpenBlocksSecondProbe(t *testing.T) {
	now := time.Now()
	b := newBreaker()
	b.nowFunc = func() time.Time { return now }

	// Trip to Open
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}

	// Advance past open duration -> first allow transitions to HalfOpen
	now = now.Add(defaultOpenDuration + time.Second)
	if err := b.allow(); err != nil {
		t.Fatalf("expected first probe to be allowed, got %v", err)
	}

	// Second call in HalfOpen should be rejected
	err := b.allow()
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected second probe to be rejected with ErrCircuitOpen, got %v", err)
	}
}

func TestBreakerRecordSuccessInClosedIsNoOp(t *testing.T) {
	b := newBreaker()

	// Recording success in Closed state should not change state
	b.recordSuccess()
	if b.getState() != StateClosed {
		t.Errorf("expected StateClosed after recordSuccess in Closed, got %v", b.getState())
	}
	if b.consecutiveFails != 0 {
		t.Errorf("expected consecutiveFails to be 0, got %d", b.consecutiveFails)
	}
}

func TestBreakerRecordFailureInOpenStaysOpen(t *testing.T) {
	now := time.Now()
	b := newBreaker()
	b.nowFunc = func() time.Time { return now }

	// Trip to Open
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}
	if b.getState() != StateOpen {
		t.Fatalf("expected StateOpen, got %v", b.getState())
	}

	// Record additional failure while Open — should stay Open, not reset timer
	openedAt := b.openedAt
	b.recordFailure()
	if b.getState() != StateOpen {
		t.Errorf("expected StateOpen after extra failure, got %v", b.getState())
	}
	if b.openedAt != openedAt {
		t.Errorf("expected openedAt unchanged, was %v, now %v", openedAt, b.openedAt)
	}
}

func TestBreakerGetStateIsReadOnly(t *testing.T) {
	now := time.Now()
	b := newBreaker()
	b.nowFunc = func() time.Time { return now }

	// Trip to Open
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}
	if b.getState() != StateOpen {
		t.Fatalf("expected StateOpen, got %v", b.getState())
	}

	// After timeout: getState should still report Open (read-only, no side effect)
	now = now.Add(defaultOpenDuration + time.Second)
	if b.getState() != StateOpen {
		t.Errorf("expected getState to remain StateOpen (read-only), got %v", b.getState())
	}

	// allow() is the sole mutator — it transitions Open -> HalfOpen and allows the probe
	if err := b.allow(); err != nil {
		t.Errorf("expected probe to be allowed, got %v", err)
	}
	if b.getState() != StateHalfOpen {
		t.Errorf("expected StateHalfOpen after allow(), got %v", b.getState())
	}
}

func TestBreakerAllowBeforeTimeoutStaysOpen(t *testing.T) {
	now := time.Now()
	b := newBreaker()
	b.nowFunc = func() time.Time { return now }

	// Trip to Open
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = b.allow()
		b.recordFailure()
	}

	// Before timeout: allow() should reject
	now = now.Add(defaultOpenDuration - time.Second)
	err := b.allow()
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen before timeout, got %v", err)
	}
}

func TestBreakerConcurrentAccess(t *testing.T) {
	b := newBreaker()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.allow()
			b.recordFailure()
			b.recordSuccess()
			b.getState()
		}()
	}
	wg.Wait()
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("State(%d).String() = %q, expected %q", tt.state, got, tt.expected)
			}
		})
	}
}
