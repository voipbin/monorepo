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
	if b.getState() != StateHalfOpen {
		t.Errorf("expected StateHalfOpen, got %v", b.getState())
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
