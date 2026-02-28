# Circuit Breaker Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a per-target-service circuit breaker to the inter-service RPC layer in `bin-common-handler`, giving all 30+ services fast-fail protection with zero per-service code changes.

**Architecture:** A new `circuitbreakerhandler` package implements the Closed/Open/Half-Open state machine. It is integrated into `requestHandler.sendRequest()` which is the single RPC chokepoint. Each target queue gets its own independent breaker, created lazily on first request.

**Tech Stack:** Go standard library only (`sync`, `time`, `errors`). Prometheus client for metrics. No new external dependencies.

---

### Task 1: Create the breaker state machine (`breaker.go`)

**Files:**
- Create: `bin-common-handler/pkg/circuitbreakerhandler/breaker.go`

**Step 1: Write the breaker implementation**

Create the individual circuit breaker state machine with three states (Closed, Open, Half-Open). Uses `sync.Mutex` for thread safety. Tracks consecutive failures and transitions between states based on thresholds and timeouts.

```go
package circuitbreakerhandler

import (
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota // normal operation
	StateOpen                  // rejecting requests
	StateHalfOpen              // allowing probe requests
)

// String returns the string representation of the state.
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

// breaker is an individual circuit breaker for a single target.
type breaker struct {
	mu sync.Mutex

	state            State
	consecutiveFails int
	lastFailTime     time.Time
	openedAt         time.Time

	failureThreshold int
	openDuration     time.Duration

	// clock is used for testing time-dependent behavior
	nowFunc func() time.Time
}

// newBreaker creates a new circuit breaker with default settings.
func newBreaker() *breaker {
	return &breaker{
		state:            StateClosed,
		failureThreshold: defaultFailureThreshold,
		openDuration:     defaultOpenDuration,
		nowFunc:          time.Now,
	}
}

// allow checks whether a request is allowed through the circuit breaker.
// Returns nil if allowed, ErrCircuitOpen if not.
func (b *breaker) allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return nil

	case StateOpen:
		// check if open duration has elapsed
		if b.nowFunc().Sub(b.openedAt) >= b.openDuration {
			b.state = StateHalfOpen
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// only one probe is allowed — reject additional requests while probing
		// The first caller that transitions from Open to HalfOpen gets through.
		// Subsequent callers while still in HalfOpen are rejected.
		return ErrCircuitOpen

	default:
		return nil
	}
}

// recordSuccess records a successful request.
func (b *breaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFails = 0

	switch b.state {
	case StateHalfOpen:
		b.state = StateClosed
	}
}

// recordFailure records a failed request.
func (b *breaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFails++
	b.lastFailTime = b.nowFunc()

	switch b.state {
	case StateClosed:
		if b.consecutiveFails >= b.failureThreshold {
			b.state = StateOpen
			b.openedAt = b.nowFunc()
		}

	case StateHalfOpen:
		// probe failed, go back to open
		b.state = StateOpen
		b.openedAt = b.nowFunc()
	}
}

// getState returns the current state of the breaker.
func (b *breaker) getState() State {
	b.mu.Lock()
	defer b.mu.Unlock()

	// check for implicit transition from Open to HalfOpen
	if b.state == StateOpen && b.nowFunc().Sub(b.openedAt) >= b.openDuration {
		b.state = StateHalfOpen
	}

	return b.state
}
```

**Step 2: Run test to verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && go build ./pkg/circuitbreakerhandler/`
Expected: Build succeeds (will fail until option.go with constants is created — do that in next task)

---

### Task 2: Create configuration constants (`option.go`)

**Files:**
- Create: `bin-common-handler/pkg/circuitbreakerhandler/option.go`

**Step 1: Write the constants and error**

```go
package circuitbreakerhandler

import (
	"errors"
	"time"
)

const (
	// defaultFailureThreshold is the number of consecutive failures before opening the circuit.
	defaultFailureThreshold = 5

	// defaultOpenDuration is how long the circuit stays open before transitioning to half-open.
	defaultOpenDuration = 30 * time.Second
)

// ErrCircuitOpen is returned when the circuit breaker is open for a target.
var ErrCircuitOpen = errors.New("circuit breaker is open")
```

**Step 2: Run build to verify both files compile**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && go build ./pkg/circuitbreakerhandler/`
Expected: Build succeeds

---

### Task 3: Write breaker unit tests (`breaker_test.go`)

**Files:**
- Create: `bin-common-handler/pkg/circuitbreakerhandler/breaker_test.go`

**Step 1: Write breaker state machine tests**

All tests use a controllable `nowFunc` for deterministic time behavior.

```go
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
		b.allow()
		b.recordFailure()
	}
	if b.getState() != StateOpen {
		t.Errorf("expected StateOpen, got %v", b.getState())
	}
}

func TestBreakerRejectsWhenOpen(t *testing.T) {
	b := newBreaker()
	for i := 0; i < defaultFailureThreshold; i++ {
		b.allow()
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
		b.allow()
		b.recordFailure()
	}

	// advance time past open duration
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
		b.allow()
		b.recordFailure()
	}

	// advance time to transition to half-open
	now = now.Add(defaultOpenDuration + time.Second)

	// probe request allowed (transitions Open -> HalfOpen inside allow())
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
		b.allow()
		b.recordFailure()
	}

	// advance time to transition to half-open
	now = now.Add(defaultOpenDuration + time.Second)

	// probe request allowed
	b.allow()
	b.recordFailure()

	if b.getState() != StateOpen {
		t.Errorf("expected StateOpen after failed probe, got %v", b.getState())
	}
}

func TestBreakerSuccessResetsFailureCount(t *testing.T) {
	b := newBreaker()

	// record failures just below threshold
	for i := 0; i < defaultFailureThreshold-1; i++ {
		b.allow()
		b.recordFailure()
	}

	// success resets counter
	b.recordSuccess()

	// another failure should NOT trip the breaker
	b.allow()
	b.recordFailure()

	if b.getState() != StateClosed {
		t.Errorf("expected StateClosed after reset, got %v", b.getState())
	}
}

func TestBreakerDoesNotTripBelowThreshold(t *testing.T) {
	b := newBreaker()
	for i := 0; i < defaultFailureThreshold-1; i++ {
		b.allow()
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

	// run concurrent allow/record calls to verify no race conditions
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.allow()
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
```

**Step 2: Run breaker tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && go test -v -race ./pkg/circuitbreakerhandler/`
Expected: All tests PASS, including the race detector check

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker
git add bin-common-handler/pkg/circuitbreakerhandler/
git commit -m "NOJIRA-add-circuit-breaker

Add circuitbreakerhandler package with breaker state machine

- bin-common-handler: Add breaker.go with Closed/Open/HalfOpen state machine
- bin-common-handler: Add option.go with default constants and ErrCircuitOpen
- bin-common-handler: Add breaker_test.go with comprehensive state transition tests"
```

---

### Task 4: Create the circuit breaker handler registry (`main.go`)

**Files:**
- Create: `bin-common-handler/pkg/circuitbreakerhandler/main.go`

**Step 1: Write the registry implementation**

The `CircuitBreakerHandler` manages a map of per-target breakers. Breakers are created lazily. Prometheus metrics are registered for state transitions and rejections.

```go
package circuitbreakerhandler

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// CircuitBreakerHandler manages circuit breakers for multiple targets.
type CircuitBreakerHandler interface {
	// Allow checks if a request to the target is allowed.
	// Returns nil if allowed, ErrCircuitOpen if the circuit is open.
	Allow(target string) error

	// RecordSuccess records a successful request to the target.
	RecordSuccess(target string)

	// RecordFailure records a failed request to the target.
	RecordFailure(target string)
}

type circuitBreakerHandler struct {
	mu       sync.Mutex
	breakers map[string]*breaker

	promStateTransitions *prometheus.CounterVec
	promState            *prometheus.GaugeVec
	promRejected         *prometheus.CounterVec
}

// NewCircuitBreakerHandler creates a new CircuitBreakerHandler with Prometheus metrics.
func NewCircuitBreakerHandler(namespace string) CircuitBreakerHandler {
	h := &circuitBreakerHandler{
		breakers: make(map[string]*breaker),
	}

	h.promStateTransitions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_state_transitions_total",
			Help:      "Total number of circuit breaker state transitions.",
		},
		[]string{"target", "from", "to"},
	)

	h.promState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_state",
			Help:      "Current circuit breaker state per target (0=closed, 1=half-open, 2=open).",
		},
		[]string{"target"},
	)

	h.promRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_rejected_total",
			Help:      "Total number of requests rejected by the circuit breaker.",
		},
		[]string{"target"},
	)

	prometheus.MustRegister(
		h.promStateTransitions,
		h.promState,
		h.promRejected,
	)

	return h
}

// getOrCreateBreaker returns the breaker for the target, creating one if it does not exist.
func (h *circuitBreakerHandler) getOrCreateBreaker(target string) *breaker {
	h.mu.Lock()
	defer h.mu.Unlock()

	b, ok := h.breakers[target]
	if !ok {
		b = newBreaker()
		h.breakers[target] = b
	}
	return b
}

// Allow checks if a request to the target is allowed.
func (h *circuitBreakerHandler) Allow(target string) error {
	b := h.getOrCreateBreaker(target)

	prevState := b.getState()
	err := b.allow()
	newState := b.getState()

	// track state transitions
	if prevState != newState {
		h.promStateTransitions.WithLabelValues(target, prevState.String(), newState.String()).Inc()
		h.promState.WithLabelValues(target).Set(float64(newState))
		log.Warnf("Circuit breaker state changed for target %s: %s -> %s", target, prevState.String(), newState.String())
	}

	if err != nil {
		h.promRejected.WithLabelValues(target).Inc()
		return fmt.Errorf("%w for target: %s", ErrCircuitOpen, target)
	}

	return nil
}

// RecordSuccess records a successful request to the target.
func (h *circuitBreakerHandler) RecordSuccess(target string) {
	b := h.getOrCreateBreaker(target)

	prevState := b.getState()
	b.recordSuccess()
	newState := b.getState()

	if prevState != newState {
		h.promStateTransitions.WithLabelValues(target, prevState.String(), newState.String()).Inc()
		h.promState.WithLabelValues(target).Set(float64(newState))
		log.Warnf("Circuit breaker state changed for target %s: %s -> %s", target, prevState.String(), newState.String())
	}
}

// RecordFailure records a failed request to the target.
func (h *circuitBreakerHandler) RecordFailure(target string) {
	b := h.getOrCreateBreaker(target)

	prevState := b.getState()
	b.recordFailure()
	newState := b.getState()

	if prevState != newState {
		h.promStateTransitions.WithLabelValues(target, prevState.String(), newState.String()).Inc()
		h.promState.WithLabelValues(target).Set(float64(newState))
		log.Warnf("Circuit breaker state changed for target %s: %s -> %s", target, prevState.String(), newState.String())
	}
}
```

**Step 2: Run build to verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && go build ./pkg/circuitbreakerhandler/`
Expected: Build succeeds

---

### Task 5: Write registry unit tests (`main_test.go`)

**Files:**
- Create: `bin-common-handler/pkg/circuitbreakerhandler/main_test.go`

**Step 1: Write registry tests**

These tests use `prometheus.NewPedanticRegistry()` to avoid global metric collisions with the real metrics from other tests.

**IMPORTANT: Prometheus metric registration conflict.** The `NewCircuitBreakerHandler` function calls `prometheus.MustRegister()` on the global registry. Running multiple test functions that each call `NewCircuitBreakerHandler` will panic because metrics are already registered. To avoid this:
- Use a single package-level test handler, OR
- Refactor `NewCircuitBreakerHandler` to accept a `prometheus.Registerer` parameter for testing.

The cleaner approach: add a `registerer` field and use `prometheus.DefaultRegisterer` in production. For tests, pass `prometheus.NewPedanticRegistry()`.

Update `main.go` to add an internal constructor that accepts a registerer:

Add to `main.go`:
```go
// newCircuitBreakerHandlerWithRegisterer is for testing — allows injecting a custom Prometheus registry.
func newCircuitBreakerHandlerWithRegisterer(namespace string, registerer prometheus.Registerer) CircuitBreakerHandler {
	h := &circuitBreakerHandler{
		breakers: make(map[string]*breaker),
	}

	h.promStateTransitions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_state_transitions_total",
			Help:      "Total number of circuit breaker state transitions.",
		},
		[]string{"target", "from", "to"},
	)

	h.promState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_state",
			Help:      "Current circuit breaker state per target (0=closed, 1=half-open, 2=open).",
		},
		[]string{"target"},
	)

	h.promRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_rejected_total",
			Help:      "Total number of requests rejected by the circuit breaker.",
		},
		[]string{"target"},
	)

	registerer.MustRegister(
		h.promStateTransitions,
		h.promState,
		h.promRejected,
	)

	return h
}
```

And update `NewCircuitBreakerHandler`:
```go
func NewCircuitBreakerHandler(namespace string) CircuitBreakerHandler {
	return newCircuitBreakerHandlerWithRegisterer(namespace, prometheus.DefaultRegisterer)
}
```

Now write the tests:

```go
package circuitbreakerhandler

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func newTestHandler() CircuitBreakerHandler {
	reg := prometheus.NewPedanticRegistry()
	return newCircuitBreakerHandlerWithRegisterer("test", reg)
}

func TestHandlerAllowNewTarget(t *testing.T) {
	h := newTestHandler()
	if err := h.Allow("target-a"); err != nil {
		t.Errorf("expected nil for new target, got %v", err)
	}
}

func TestHandlerIndependentBreakers(t *testing.T) {
	h := newTestHandler()

	// trip breaker for target-a
	for i := 0; i < defaultFailureThreshold; i++ {
		h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	// target-a should be open
	err := h.Allow("target-a")
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen for target-a, got %v", err)
	}

	// target-b should still be allowed
	if err := h.Allow("target-b"); err != nil {
		t.Errorf("expected nil for target-b, got %v", err)
	}
}

func TestHandlerRecordSuccessResets(t *testing.T) {
	h := newTestHandler()

	// record failures below threshold
	for i := 0; i < defaultFailureThreshold-1; i++ {
		h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	// success resets
	h.RecordSuccess("target-a")

	// more failures below threshold should not trip
	for i := 0; i < defaultFailureThreshold-1; i++ {
		h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	if err := h.Allow("target-a"); err != nil {
		t.Errorf("expected nil after reset, got %v", err)
	}
}

func TestHandlerErrorWrapsTarget(t *testing.T) {
	h := newTestHandler()

	for i := 0; i < defaultFailureThreshold; i++ {
		h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	err := h.Allow("target-a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected error to wrap ErrCircuitOpen, got %v", err)
	}

	expected := "circuit breaker is open for target: target-a"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestHandlerLazyCreation(t *testing.T) {
	h := newTestHandler()
	cbh := h.(*circuitBreakerHandler)

	if len(cbh.breakers) != 0 {
		t.Errorf("expected 0 breakers initially, got %d", len(cbh.breakers))
	}

	h.Allow("target-a")
	h.Allow("target-b")

	if len(cbh.breakers) != 2 {
		t.Errorf("expected 2 breakers after two targets, got %d", len(cbh.breakers))
	}

	// calling again should not create a new breaker
	h.Allow("target-a")
	if len(cbh.breakers) != 2 {
		t.Errorf("expected still 2 breakers, got %d", len(cbh.breakers))
	}
}
```

**Step 2: Run all circuitbreakerhandler tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && go test -v -race ./pkg/circuitbreakerhandler/`
Expected: All tests PASS

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker
git add bin-common-handler/pkg/circuitbreakerhandler/
git commit -m "NOJIRA-add-circuit-breaker

Add CircuitBreakerHandler registry with Prometheus metrics

- bin-common-handler: Add main.go with CircuitBreakerHandler interface and lazy breaker registry
- bin-common-handler: Add Prometheus metrics for state transitions, current state, and rejected requests
- bin-common-handler: Add main_test.go with registry tests (independent breakers, lazy creation, error wrapping)"
```

---

### Task 6: Integrate circuit breaker into requestHandler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:1308-1329` (struct + constructor)
- Modify: `bin-common-handler/pkg/requesthandler/send_request.go:27-57` (sendRequest function)

**Step 1: Add `cb` field to `requestHandler` struct**

In `bin-common-handler/pkg/requesthandler/main.go`, modify the struct at line 1308:

```go
type requestHandler struct {
	sock sockhandler.SockHandler

	publisher commonoutline.ServiceName

	utilHandler utilhandler.UtilHandler

	cb circuitbreakerhandler.CircuitBreakerHandler
}
```

Add import for `"monorepo/bin-common-handler/pkg/circuitbreakerhandler"`.

**Step 2: Initialize `cb` in `NewRequestHandler`**

In `bin-common-handler/pkg/requesthandler/main.go`, modify the constructor at line 1317:

```go
func NewRequestHandler(sock sockhandler.SockHandler, publisher commonoutline.ServiceName) RequestHandler {
	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	h := &requestHandler{
		sock: sock,

		publisher:   publisher,
		utilHandler: utilhandler.NewUtilHandler(),
		cb:          circuitbreakerhandler.NewCircuitBreakerHandler(namespace),
	}

	return h
}
```

Note: `initPrometheus(namespace)` is moved before the struct initialization to keep the namespace computation clean. The namespace variable was already computed before `initPrometheus` in the original code.

**Step 3: Add circuit breaker check to `sendRequest`**

In `bin-common-handler/pkg/requesthandler/send_request.go`, modify the `sendRequest` function. The `default` case (direct requests) gets wrapped:

```go
func (r *requestHandler) sendRequest(ctx context.Context, queue commonoutline.QueueName, uri string, method sock.RequestMethod, resource string, timeout int, delay int, dataType string, data json.RawMessage) (*sock.Response, error) {
	// creat a request message
	req := &sock.Request{
		URI:       uri,
		Method:    method,
		Publisher: string(r.publisher),
		DataType:  dataType,
		Data:      data,
	}

	cctx, cancel := context.WithTimeout(ctx, time.Millisecond*time.Duration(timeout))
	defer cancel()

	switch {
	case delay > 0:
		// send scheduled message.
		// we don't expect the response message here.
		if err := r.sendDelayedRequest(string(queue), resource, delay, req); err != nil {
			return nil, errors.Wrapf(err, "could not send the delayed request. queue: %s, method: %s, uri: %s", queue, method, uri)
		}
		return nil, nil

	default:
		// check circuit breaker before sending
		if err := r.cb.Allow(string(queue)); err != nil {
			return nil, errors.Wrapf(err, "could not send the request. queue: %s, method: %s, uri: %s", queue, method, uri)
		}

		res, err := r.sendDirectRequest(cctx, string(queue), resource, req)
		if err != nil {
			r.cb.RecordFailure(string(queue))
			return nil, errors.Wrapf(err, "could not send the request. queue: %s, method: %s, uri: %s", queue, method, uri)
		}

		r.cb.RecordSuccess(string(queue))
		return res, nil
	}
}
```

**Step 4: Run build to verify compilation**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && go build ./pkg/requesthandler/`
Expected: Build succeeds

**Step 5: Run the mock generation**

The `requesthandler/main.go` has `//go:generate mockgen ...`. After modifying the struct, regenerate mocks:

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && go generate ./pkg/requesthandler/`
Expected: `mock_main.go` regenerated

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker
git add bin-common-handler/pkg/requesthandler/main.go bin-common-handler/pkg/requesthandler/send_request.go bin-common-handler/pkg/requesthandler/mock_main.go
git commit -m "NOJIRA-add-circuit-breaker

Integrate circuit breaker into requestHandler sendRequest

- bin-common-handler: Add cb field to requestHandler struct
- bin-common-handler: Initialize CircuitBreakerHandler in NewRequestHandler
- bin-common-handler: Wrap sendDirectRequest with Allow/RecordSuccess/RecordFailure
- bin-common-handler: Delayed requests bypass circuit breaker (fire-and-forget)"
```

---

### Task 7: Generate mock for CircuitBreakerHandler

**Files:**
- Modify: `bin-common-handler/pkg/circuitbreakerhandler/main.go` (add go:generate directive)
- Create: `bin-common-handler/pkg/circuitbreakerhandler/mock_main.go` (generated)

**Step 1: Add go:generate directive to `main.go`**

Add at the top of `bin-common-handler/pkg/circuitbreakerhandler/main.go`:

```go
//go:generate mockgen -package circuitbreakerhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
```

**Step 2: Run mock generation**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && go generate ./pkg/circuitbreakerhandler/`
Expected: `mock_main.go` created

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker
git add bin-common-handler/pkg/circuitbreakerhandler/
git commit -m "NOJIRA-add-circuit-breaker

Add mock generation for CircuitBreakerHandler interface

- bin-common-handler: Add go:generate directive and generated mock_main.go"
```

---

### Task 8: Run full verification workflow for bin-common-handler

**Files:**
- No new files — verification only

**Step 1: Run the full verification workflow**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-common-handler && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass. If lint warnings appear, fix them before proceeding.

**Step 2: Fix any issues and re-run**

If tests or linting fail, fix the issues and re-run the verification workflow. Common issues:
- Unused imports
- Missing error handling
- Lint warnings about mutex copying (use pointer receivers consistently)

**Step 3: Commit any fixes**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker
git add bin-common-handler/
git commit -m "NOJIRA-add-circuit-breaker

Fix lint and verification issues for circuit breaker

- bin-common-handler: Address any lint warnings or test failures"
```

---

### Task 9: Verify downstream services build (spot check)

**Files:**
- No new files — verification only

**Step 1: Pick 2-3 representative services and run their verification**

Since `bin-common-handler` changed, verify that downstream services still compile. Pick services that heavily use `requesthandler`:

Run (sequentially):
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-call-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-flow-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker/bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All three pass. Since `NewRequestHandler()` signature is unchanged, no code changes should be needed in downstream services.

**Step 2: If any downstream service fails, investigate and fix**

The most likely issue is vendor directory needing updates. The `go mod tidy && go mod vendor` step should handle this.

---

### Task 10: Final commit and summary

**Step 1: Review all changes**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker && git log --oneline`
Expected: 3-4 clean commits on the branch

**Step 2: Push branch**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker && git push -u origin NOJIRA-add-circuit-breaker`

**Step 3: Create PR**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-circuit-breaker && \
gh pr create --title "NOJIRA-add-circuit-breaker" --body "$(cat <<'EOF'
Add circuit breaker for inter-service RPC to protect all 30+ services from
cascading failures when a downstream service becomes unavailable.

- bin-common-handler: Add circuitbreakerhandler package with Closed/Open/HalfOpen state machine
- bin-common-handler: Add CircuitBreakerHandler registry with lazy per-target breaker creation
- bin-common-handler: Add Prometheus metrics (state transitions, current state, rejected requests)
- bin-common-handler: Integrate circuit breaker into requestHandler.sendRequest()
- bin-common-handler: Delayed (fire-and-forget) requests bypass circuit breaker
- bin-common-handler: Zero per-service code changes required (constructor signature unchanged)
EOF
)"
```
