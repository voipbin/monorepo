package k8swatchhandler

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// stubResourceVersion is a resourceVersionReporter whose value the test controls.
//
// The real signal it stands in for is a reflector completing a list/watch, which bumps the
// informer's last-synced resource version. There is no way to force that on a live informer
// deterministically, which is exactly why watchUntilDone takes the narrow interface.
type stubResourceVersion struct {
	mu      sync.Mutex
	version string
}

func (s *stubResourceVersion) LastSyncResourceVersion() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.version
}

func (s *stubResourceVersion) set(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.version = version
}

func testLogEntry() *logrus.Entry {
	return logrus.WithField("func", "test")
}

// newWatchLoopHandler builds a handler tuned for fast watch-loop tests.
func newWatchLoopHandler(threshold int) *k8sWatchHandler {
	return &k8sWatchHandler{
		cacheSyncTimeout:    time.Second,
		watchHealthInterval: 5 * time.Millisecond,
		maxWatchFailures:    threshold,
	}
}

// Test_watchUntilDone_resourceVersionChangeResetsTheBudget is the regression test for the reset
// path a code review found completely uncovered: replacing its RecordHealthy call with a no-op
// left every other test in this package green.
//
// The reset matters because delivered events are NOT a sufficient recovery signal. A selector
// matching zero pods delivers nothing however healthy the watch is, so a cluster with no Asterisk
// pods yet would drain the budget on routine apiserver blips and restart a working process. A
// changed resource version proves the reflector completed a list/watch regardless of whether any
// object matched — this test therefore delivers ZERO events on purpose.
func Test_watchUntilDone_resourceVersionChangeResetsTheBudget(t *testing.T) {
	h := newWatchLoopHandler(10)

	budget := newWatchFailureBudget(h.maxWatchFailures, nil)
	informer := &stubResourceVersion{version: "100"}
	done := make(chan struct{})

	// spend part of the budget, as a run of transient watch errors would.
	for i := 0; i < 3; i++ {
		budget.RecordFailure()
	}
	if budget.Consecutive() != 3 {
		t.Fatalf("Wrong match. expect: 3 consecutive failures, got: %d", budget.Consecutive())
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- h.watchUntilDone(ctx, informer, budget, done, testLogEntry()) }()

	// wait for the baseline read: watchUntilDone captures LastSyncResourceVersion once on entry,
	// so changing it before that read would make the "change" invisible and the test would be
	// asserting nothing.
	time.Sleep(40 * time.Millisecond)
	if budget.Consecutive() != 3 {
		t.Fatalf("Wrong match. expect: the budget still at 3 before the version changes, got: %d", budget.Consecutive())
	}

	// the reflector completes a relist; no event is ever delivered.
	informer.set("200")

	deadline := time.After(3 * time.Second)
	for budget.Consecutive() != 0 {
		select {
		case <-deadline:
			t.Fatalf("Wrong match. expect: the budget to reset on a resource version change, got: %d consecutive", budget.Consecutive())
		case <-time.After(2 * time.Millisecond):
		}
	}

	cancel()
	close(done)

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("Wrong match. expect: nil on graceful shutdown, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("Wrong match. expect: watchUntilDone to return, got: still running")
	}
}

// Test_watchUntilDone_unchangedResourceVersionDoesNotReset pins the other half: the reset is tied
// to an actual CHANGE, not merely to the ticker firing. A tick-driven unconditional reset would
// make the budget unable to ever exhaust, silently disabling the fail-loud path.
func Test_watchUntilDone_unchangedResourceVersionDoesNotReset(t *testing.T) {
	h := newWatchLoopHandler(10)

	budget := newWatchFailureBudget(h.maxWatchFailures, nil)
	informer := &stubResourceVersion{version: "100"}
	done := make(chan struct{})

	for i := 0; i < 3; i++ {
		budget.RecordFailure()
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- h.watchUntilDone(ctx, informer, budget, done, testLogEntry()) }()

	// let many ticks elapse with the resource version held constant.
	time.Sleep(60 * time.Millisecond)

	if got := budget.Consecutive(); got != 3 {
		t.Errorf("Wrong match. expect: the budget untouched at 3, got: %d", got)
	}

	cancel()
	close(done)
	<-result
}

// Test_watchUntilDone_budgetExhaustionReturnsError pins the fail-loud branch: an exhausted budget
// must surface as an error so cmd/sentinel-manager exits non-zero and Komodo shows a crash-loop.
func Test_watchUntilDone_budgetExhaustionReturnsError(t *testing.T) {
	h := newWatchLoopHandler(2)

	budget := newWatchFailureBudget(h.maxWatchFailures, nil)
	informer := &stubResourceVersion{version: "100"}
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() { result <- h.watchUntilDone(ctx, informer, budget, done, testLogEntry()) }()

	budget.RecordFailure()
	budget.RecordFailure()

	select {
	case err := <-result:
		if err == nil {
			t.Fatalf("Wrong match. expect: error on budget exhaustion, got: nil")
		}
		if !strings.Contains(err.Error(), "consecutive times without recovering") {
			t.Errorf("Wrong match. expect: an exhaustion error, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("Wrong match. expect: watchUntilDone to return an error, got: still running")
	}
}

// Test_watchUntilDone_informerStoppingWithLiveContextIsAnError pins the third branch. An informer
// that stops on its own while the service is supposed to be running means coverage silently
// dropped to nothing for that selector — precisely the "looks up, watches nothing" state.
func Test_watchUntilDone_informerStoppingWithLiveContextIsAnError(t *testing.T) {
	h := newWatchLoopHandler(10)

	budget := newWatchFailureBudget(h.maxWatchFailures, nil)
	informer := &stubResourceVersion{version: "100"}
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() { result <- h.watchUntilDone(ctx, informer, budget, done, testLogEntry()) }()

	// the informer stops while the context is still very much alive.
	close(done)

	select {
	case err := <-result:
		if err == nil {
			t.Fatalf("Wrong match. expect: error when the informer stops unexpectedly, got: nil")
		}
		if !strings.Contains(err.Error(), "stopped unexpectedly") {
			t.Errorf("Wrong match. expect: a stopped-unexpectedly error, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("Wrong match. expect: watchUntilDone to return an error, got: still running")
	}
}

// Test_watchUntilDone_contextCancellationIsGraceful pins that the same informer stop is NOT an
// error once the context is cancelled — that is an ordinary shutdown and must exit zero.
func Test_watchUntilDone_contextCancellationIsGraceful(t *testing.T) {
	h := newWatchLoopHandler(10)

	budget := newWatchFailureBudget(h.maxWatchFailures, nil)
	informer := &stubResourceVersion{version: "100"}
	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())

	result := make(chan error, 1)
	go func() { result <- h.watchUntilDone(ctx, informer, budget, done, testLogEntry()) }()

	cancel()
	close(done)

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("Wrong match. expect: nil on graceful shutdown, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("Wrong match. expect: watchUntilDone to return, got: still running")
	}
}

// Test_watchUntilDone_shutdownWinsOverBudgetExhaustion pins the ctx-priority guard.
//
// Go's select picks uniformly at random among ready cases, so with both a cancelled context and an
// exhausted budget ready, the naive form would report a failure and exit non-zero roughly half the
// time during an ordinary shutdown. Repeated because a single pass would pass by luck.
func Test_watchUntilDone_shutdownWinsOverBudgetExhaustion(t *testing.T) {
	for i := 0; i < 50; i++ {
		h := newWatchLoopHandler(1)

		budget := newWatchFailureBudget(h.maxWatchFailures, nil)
		informer := &stubResourceVersion{version: "100"}
		done := make(chan struct{})

		ctx, cancel := context.WithCancel(context.Background())

		// both conditions are already true before the loop starts.
		budget.RecordFailure()
		cancel()
		close(done)

		if err := h.watchUntilDone(ctx, informer, budget, done, testLogEntry()); err != nil {
			t.Fatalf("Wrong match at iteration %d. expect: nil when shutdown coincides with exhaustion, got: %v", i, err)
		}
	}
}
