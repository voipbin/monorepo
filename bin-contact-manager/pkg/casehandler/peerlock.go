package casehandler

import (
	"context"
	"fmt"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

// peerLockTimeout bounds how long GetOrCreate will wait to acquire the
// per-peer-tuple in-process serialization lock before giving up (VOIP-1232
// design v6, round 5's mandate). An unbounded wait, combined with
// subscribehandler's per-message unbounded-goroutine dispatch
// (processEventRun's `go h.processEvent(m)`), risks goroutine pileup under
// a sustained hot-peer burst. This timeout is a self-contained
// context.WithTimeout wait -- it fires purely on wall-clock elapsed time
// and has no dependency on what the current lock holder's own retry loop
// is doing (round-5's identified flaw in an earlier design revision that
// tried to bound acquisition via the deadlock-retry loop's attempt count).
//
// 5s was chosen with wide margin over the expected lock-hold duration: a
// held GetOrCreate call runs its own maxDeadlockRetries (outer) x
// maxInsertRetries (inner) bounded retry loops, expected well under 1-2s
// total at normal same-DC MySQL round-trip latency under contact-manager's
// current single-replica deployment (no cross-pod contention to inflate
// this further).
const peerLockTimeout = 5 * time.Second

// ErrPeerLockTimeout is returned when GetOrCreate could not acquire the
// per-peer-tuple in-process lock within peerLockTimeout. Distinct from
// ErrDeadlockExhausted and ErrGetOrCreateExhausted so callers
// (subscribehandler.processEvent) can tag it separately for interim
// triage (VOIP-1232 design v6 item 7) -- this failure currently still
// falls through the same ack-before-process silent-drop pipeline as any
// other GetOrCreate error (tracked separately in VOIP-1233).
var ErrPeerLockTimeout = fmt.Errorf("could not acquire peer serialization lock within timeout")

var (
	promPeerLockMapSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "case_peer_lock_map_size",
			Help:      "Current number of distinct peer-tuple entries held in the in-process GetOrCreate serialization lock map",
		},
	)

	promDeadlockRetryTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "case_getorcreate_deadlock_retry_total",
			Help:      "Total number of GetOrCreate attempts restarted after a MySQL deadlock (errno 1213)",
		},
	)

	promDeadlockExhaustedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "case_getorcreate_deadlock_exhausted_total",
			Help:      "Total number of GetOrCreate calls that exhausted all deadlock retry attempts and gave up",
		},
	)

	promPeerLockTimeoutTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "case_getorcreate_peer_lock_timeout_total",
			Help:      "Total number of GetOrCreate calls that failed to acquire the per-peer serialization lock within the timeout",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promPeerLockMapSize,
		promDeadlockRetryTotal,
		promDeadlockExhaustedTotal,
		promPeerLockTimeoutTotal,
	)
}

// metricsNamespace matches subscribehandler's own "contact_manager"
// namespace constant (kept package-local here to avoid a cross-package
// import just for a string constant; must stay in sync).
const metricsNamespace = "contact_manager"

// peerLockKey builds the map key for a (customer_id, peer_type,
// peer_target, reference_type) tuple -- the same tuple that identifies an
// OPEN Case under uq_case_open_peer (design §3.1/§4).
func peerLockKey(customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) string {
	return customerID.String() + "|" + string(peerType) + "|" + peerTarget + "|" + referenceType
}

// acquirePeerLock acquires (lazily creating on first use) the buffered-
// channel semaphore for the given tuple key, waiting up to
// peerLockTimeout. Returns a release func that MUST be called exactly
// once when the caller is done (typically via a deferred call, or an
// explicit call right after tx.Commit() per GetOrCreate's release-timing
// requirement -- see GetOrCreate's call site comment).
//
// VOIP-1232 design v6: this in-process lock serializes concurrent
// GetOrCreate calls for the SAME peer tuple within a single contact-
// manager process, eliminating the same-tuple concurrent-INSERT race that
// produces MySQL deadlocks (1213) at the source, rather than only
// absorbing it via DB-level retry (see getorcreate.go's deadlock-retry
// loop, which remains the topology-independent correctness backstop
// regardless of replica count).
//
// TOPOLOGY CAVEAT (must be stated in the PR body per round-4/5's
// mandate): this lock is IN-PROCESS ONLY, not cross-pod/distributed. It
// fully prevents the race under contact-manager's CURRENT single-replica
// production config (confirmed replicas: 1, no HPA, in both
// bin-contact-manager/k8s/deployment.yml and the install repo's
// k8s/backend/services/contact-manager.yaml). Under a future multi-
// replica deployment, RabbitMQ's plain shared-queue competing-consumers
// model has no per-peer-tuple pod affinity, so this lock would provide
// ZERO cross-pod protection and the fix would degrade to relying on
// DB-level deadlock retry alone. A Redis-based distributed lock (contact-
// manager already depends on Redis for contact-body caching) is the
// correct upgrade path if/when replicas are ever increased -- tracked as
// a follow-up, not built preemptively here.
func (h *caseHandler) acquirePeerLock(ctx context.Context, key string) (func(), error) {
	ch := h.loadOrCreatePeerLockChan(key)

	lockCtx, cancel := context.WithTimeout(ctx, peerLockTimeout)
	defer cancel()

	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-lockCtx.Done():
		return nil, ErrPeerLockTimeout
	}
}

// loadOrCreatePeerLockChan returns the buffered channel (capacity 1) for
// key, creating it under a write lock if this is the first use.
//
// Double-checked locking, spelled out explicitly (VOIP-1232 round-5's
// flagged ambiguity): fast path takes only an RLock to reuse an existing
// entry; on a miss, it takes the write Lock and RE-CHECKS the map before
// creating -- without this re-check, two goroutines racing on first-use
// for the same brand-new key could each create and store a DISTINCT
// channel object, with the second silently clobbering the first's map
// entry. Both goroutines would then believe they've acquired mutual
// exclusion while actually holding/selecting on two different channels,
// running fully concurrently -- exactly the correctness break this whole
// mechanism exists to prevent.
func (h *caseHandler) loadOrCreatePeerLockChan(key string) chan struct{} {
	h.peerLocksMu.RLock()
	ch, ok := h.peerLocks[key]
	h.peerLocksMu.RUnlock()
	if ok {
		return ch
	}

	h.peerLocksMu.Lock()
	defer h.peerLocksMu.Unlock()
	// Defensive lazy-init: NewCaseHandler always initializes peerLocks,
	// but a caseHandler constructed directly as a struct literal (as many
	// existing unit tests do, e.g. getorcreate_race_test.go) would
	// otherwise have a nil map here, panicking on the write below.
	if h.peerLocks == nil {
		h.peerLocks = make(map[string]chan struct{})
	}
	// Re-check under the write lock: another goroutine may have created
	// this exact key between our RUnlock above and this Lock.
	if ch, ok := h.peerLocks[key]; ok {
		return ch
	}
	ch = make(chan struct{}, 1)
	h.peerLocks[key] = ch
	promPeerLockMapSize.Set(float64(len(h.peerLocks)))
	return ch
}
