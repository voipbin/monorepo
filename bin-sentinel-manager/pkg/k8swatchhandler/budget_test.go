package k8swatchhandler

import (
	"sync"
	"testing"
)

// recordOutcomes returns a budget plus a slice capturing every outcome it reports.
func recordOutcomes(threshold int) (*watchFailureBudget, *[]string, *sync.Mutex) {
	var mu sync.Mutex
	outcomes := []string{}

	budget := newWatchFailureBudget(threshold, func(outcome string) {
		mu.Lock()
		defer mu.Unlock()
		outcomes = append(outcomes, outcome)
	})

	return budget, &outcomes, &mu
}

// Test_watchFailureBudget_transientFailuresDoNotExhaust pins the reason this budget exists at all.
//
// SetWatchErrorHandler fires on entirely benign conditions — an apiserver rolling restart, a
// `too old resource version` forcing a relist, a transient connection reset. Treating any single
// invocation as fatal would self-restart a perfectly healthy cluster on the first blip.
func Test_watchFailureBudget_transientFailuresDoNotExhaust(t *testing.T) {
	budget, outcomes, mu := recordOutcomes(5)

	for i := 0; i < 4; i++ {
		if _, exhausted := budget.RecordFailure(); exhausted {
			t.Fatalf("Wrong match. expect: not exhausted at failure %d, got: exhausted", i+1)
		}
	}

	select {
	case <-budget.Fatal():
		t.Errorf("Wrong match. expect: the fatal channel open, got: closed")
	default:
	}

	mu.Lock()
	defer mu.Unlock()
	if len(*outcomes) != 4 {
		t.Fatalf("Wrong match. expect: 4 outcomes, got: %d (%v)", len(*outcomes), *outcomes)
	}
	for i, outcome := range *outcomes {
		if outcome != watchOutcomeTransientError {
			t.Errorf("Wrong match at %d. expect: %s, got: %s", i, watchOutcomeTransientError, outcome)
		}
	}
}

// Test_watchFailureBudget_exhaustionIsFatal pins the other half: a sustained run with no recovery
// must convert to a fatal signal, not an infinite quiet retry.
func Test_watchFailureBudget_exhaustionIsFatal(t *testing.T) {
	budget, outcomes, mu := recordOutcomes(3)

	// called in a loop rather than `a() || a()`: the short-circuit would skip the second call
	// the moment the first returned true, silently weakening the test.
	for i := 0; i < 2; i++ {
		if _, exhausted := budget.RecordFailure(); exhausted {
			t.Fatalf("Wrong match. expect: not exhausted at failure %d, got: exhausted", i+1)
		}
	}
	if _, exhausted := budget.RecordFailure(); !exhausted {
		t.Fatalf("Wrong match. expect: exhausted at the threshold, got: not exhausted")
	}

	select {
	case <-budget.Fatal():
	default:
		t.Errorf("Wrong match. expect: the fatal channel closed, got: open")
	}

	mu.Lock()
	defer mu.Unlock()
	last := (*outcomes)[len(*outcomes)-1]
	if last != watchOutcomeFatal {
		t.Errorf("Wrong match. expect: %s, got: %s", watchOutcomeFatal, last)
	}
}

// Test_watchFailureBudget_resetsOnHealthy pins that the budget counts CONSECUTIVE failures.
// Without the reset, a long-lived process would accumulate unrelated blips across days and
// eventually restart itself with no ongoing fault.
func Test_watchFailureBudget_resetsOnHealthy(t *testing.T) {
	budget, outcomes, mu := recordOutcomes(3)

	budget.RecordFailure()
	budget.RecordFailure()
	if budget.Consecutive() != 2 {
		t.Fatalf("Wrong match. expect: 2, got: %d", budget.Consecutive())
	}

	budget.RecordHealthy()
	if budget.Consecutive() != 0 {
		t.Fatalf("Wrong match. expect: 0 after recovery, got: %d", budget.Consecutive())
	}

	// the budget must now tolerate a full fresh run of failures.
	for i := 0; i < 2; i++ {
		if _, exhausted := budget.RecordFailure(); exhausted {
			t.Errorf("Wrong match. expect: the budget to start over after recovery, got: exhausted at %d", i+1)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	resynced := 0
	for _, outcome := range *outcomes {
		if outcome == watchOutcomeResynced {
			resynced++
		}
	}
	if resynced != 1 {
		t.Errorf("Wrong match. expect: exactly 1 resynced outcome, got: %d (%v)", resynced, *outcomes)
	}
}

// Test_watchFailureBudget_healthyWithNothingToRecoverIsQuiet pins that the common case is silent.
// RecordHealthy runs on EVERY delivered event, so reporting an outcome each time would bury the
// genuinely interesting transitions under per-event noise.
func Test_watchFailureBudget_healthyWithNothingToRecoverIsQuiet(t *testing.T) {
	budget, outcomes, mu := recordOutcomes(3)

	for i := 0; i < 100; i++ {
		budget.RecordHealthy()
	}

	mu.Lock()
	defer mu.Unlock()
	if len(*outcomes) != 0 {
		t.Errorf("Wrong match. expect: no outcomes reported, got: %d (%v)", len(*outcomes), *outcomes)
	}
}

// Test_watchFailureBudget_fatalIsTerminal pins that a late event cannot revive an exhausted
// budget. The process is already on its way out; letting a straggler reset it would leave Run's
// fatal path racing a handler that believes everything is fine.
func Test_watchFailureBudget_fatalIsTerminal(t *testing.T) {
	budget, _, _ := recordOutcomes(2)

	budget.RecordFailure()
	if _, exhausted := budget.RecordFailure(); !exhausted {
		t.Fatalf("Wrong match. expect: exhausted, got: not exhausted")
	}

	budget.RecordHealthy()

	if budget.Consecutive() == 0 {
		t.Errorf("Wrong match. expect: an exhausted budget to refuse the reset, got: reset to 0")
	}
	if _, exhausted := budget.RecordFailure(); !exhausted {
		t.Errorf("Wrong match. expect: still exhausted, got: not exhausted")
	}

	select {
	case <-budget.Fatal():
	default:
		t.Errorf("Wrong match. expect: the fatal channel to stay closed, got: open")
	}
}

// Test_watchFailureBudget_fatalChannelClosesOnce pins that repeated exhaustion does not panic on a
// double close.
func Test_watchFailureBudget_fatalChannelClosesOnce(t *testing.T) {
	budget, _, _ := recordOutcomes(1)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecordFailure panicked on repeated exhaustion: %v", r)
		}
	}()

	for i := 0; i < 10; i++ {
		budget.RecordFailure()
	}
}

// Test_watchFailureBudget_nilOutcomeCallback pins the nil-callback guard: the constructor accepts
// one, so nothing should have to check before reporting.
func Test_watchFailureBudget_nilOutcomeCallback(t *testing.T) {
	budget := newWatchFailureBudget(2, nil)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("budget panicked with a nil outcome callback: %v", r)
		}
	}()

	budget.RecordFailure()
	budget.RecordHealthy()
	budget.RecordFailure()
	budget.RecordFailure()
}

// Test_watchFailureBudget_concurrentAccess exercises the lock under -race: the watch error handler
// and the event callbacks run on different client-go goroutines.
func Test_watchFailureBudget_concurrentAccess(t *testing.T) {
	budget := newWatchFailureBudget(1000, func(string) {})

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); budget.RecordFailure() }()
		go func() { defer wg.Done(); budget.RecordHealthy() }()
		go func() { defer wg.Done(); _ = budget.Consecutive() }()
	}
	wg.Wait()
}
