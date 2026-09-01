package k8swatchhandler

import (
	"sync"
)

// watchFailureBudget converts a run of consecutive watch failures into a single fatal signal
// (design §8.4 item 3).
//
// The design decision this encodes, and the reason it is not just "log the error": client-go's
// watch error handler fires on routine, benign conditions — an apiserver rolling restart, a
// `too old resource version` forcing a relist, a transient connection reset. Wiring that straight
// to "fatal" would self-restart a perfectly healthy system on the first blip; wiring it to "just
// log" is the silent-failure behavior this whole rewrite exists to remove. Only a sustained run of
// failures with NO intervening recovery is genuinely fatal.
type watchFailureBudget struct {
	mu sync.Mutex

	threshold   int
	consecutive int
	fatal       bool

	// fatalCh closes exactly once, when the budget is exhausted, so Run can select on it.
	fatalCh   chan struct{}
	closeOnce sync.Once

	// onOutcome reports each transition. Split out as a field so tests can observe outcomes
	// without scraping Prometheus.
	onOutcome func(outcome string)
}

func newWatchFailureBudget(threshold int, onOutcome func(outcome string)) *watchFailureBudget {
	if onOutcome == nil {
		onOutcome = func(string) {}
	}

	return &watchFailureBudget{
		threshold: threshold,
		fatalCh:   make(chan struct{}),
		onOutcome: onOutcome,
	}
}

// RecordFailure registers one watch error and reports whether the budget is now exhausted.
func (h *watchFailureBudget) RecordFailure() bool {
	h.mu.Lock()

	if h.fatal {
		h.mu.Unlock()
		return true
	}

	h.consecutive++
	exhausted := h.consecutive >= h.threshold
	if exhausted {
		h.fatal = true
	}
	h.mu.Unlock()

	if exhausted {
		h.onOutcome(watchOutcomeFatal)
		h.closeOnce.Do(func() { close(h.fatalCh) })
		return true
	}

	h.onOutcome(watchOutcomeTransientError)

	return false
}

// RecordHealthy resets the budget after evidence the watch is working.
//
// It is deliberately cheap and idempotent: it is called on EVERY delivered event, so the common
// case must do nothing beyond taking the lock. The `resynced` outcome is only reported when there
// was actually something to recover from — otherwise a busy cluster would bury the genuinely
// interesting transitions under one `resynced` per event.
//
// Once the budget is exhausted the reset is refused: the process is already on its way out, and
// letting a late event revive it would leave Run's fatal path racing against a handler that
// believes everything is fine.
func (h *watchFailureBudget) RecordHealthy() {
	h.mu.Lock()

	if h.fatal || h.consecutive == 0 {
		h.mu.Unlock()
		return
	}

	h.consecutive = 0
	h.mu.Unlock()

	h.onOutcome(watchOutcomeResynced)
}

// Fatal returns the channel that closes when the budget is exhausted.
func (h *watchFailureBudget) Fatal() <-chan struct{} {
	return h.fatalCh
}

// Consecutive returns the current consecutive-failure count.
func (h *watchFailureBudget) Consecutive() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.consecutive
}
