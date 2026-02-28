package circuitbreakerhandler

import (
	"sync"
	"time"
)

type State int

const (
	StateClosed   State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type breaker struct {
	mu sync.Mutex

	state            State
	consecutiveFails int
	openedAt         time.Time

	failureThreshold int
	openDuration     time.Duration

	nowFunc func() time.Time
}

func newBreaker() *breaker {
	return &breaker{
		state:            StateClosed,
		failureThreshold: defaultFailureThreshold,
		openDuration:     defaultOpenDuration,
		nowFunc:          time.Now,
	}
}

// allow checks whether a request is allowed through.
// Returns nil if allowed, ErrCircuitOpen if not.
func (b *breaker) allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return nil
	case StateOpen:
		if b.nowFunc().Sub(b.openedAt) >= b.openDuration {
			b.state = StateHalfOpen
			return nil
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		// only one probe allowed — the caller that transitioned Open→HalfOpen
		return ErrCircuitOpen
	default:
		return nil
	}
}

func (b *breaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFails = 0
	if b.state == StateHalfOpen {
		b.state = StateClosed
	}
}

func (b *breaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFails++

	switch b.state {
	case StateClosed:
		if b.consecutiveFails >= b.failureThreshold {
			b.state = StateOpen
			b.openedAt = b.nowFunc()
		}
	case StateHalfOpen:
		b.state = StateOpen
		b.openedAt = b.nowFunc()
	}
}

func (b *breaker) getState() State {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == StateOpen && b.nowFunc().Sub(b.openedAt) >= b.openDuration {
		b.state = StateHalfOpen
	}
	return b.state
}
