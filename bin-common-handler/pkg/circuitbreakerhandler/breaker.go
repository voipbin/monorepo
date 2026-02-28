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
// Returns the state before and after the check, and an error if rejected.
// Both states are captured under a single lock acquisition to avoid TOCTOU races.
func (b *breaker) allow() (State, State, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	prev := b.state

	switch b.state {
	case StateClosed:
		return prev, b.state, nil
	case StateOpen:
		if b.nowFunc().Sub(b.openedAt) >= b.openDuration {
			b.state = StateHalfOpen
			return prev, b.state, nil
		}
		return prev, b.state, ErrCircuitOpen
	case StateHalfOpen:
		// only one probe allowed — the caller that transitioned Open→HalfOpen
		return prev, b.state, ErrCircuitOpen
	default:
		return prev, b.state, nil
	}
}

// recordSuccess records a successful request.
// Returns the state before and after, captured atomically.
func (b *breaker) recordSuccess() (State, State) {
	b.mu.Lock()
	defer b.mu.Unlock()

	prev := b.state
	b.consecutiveFails = 0
	if b.state == StateHalfOpen {
		b.state = StateClosed
	}
	return prev, b.state
}

// recordFailure records a failed request.
// Returns the state before and after, captured atomically.
func (b *breaker) recordFailure() (State, State) {
	b.mu.Lock()
	defer b.mu.Unlock()

	prev := b.state
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
	return prev, b.state
}

func (b *breaker) getState() State {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.state
}
