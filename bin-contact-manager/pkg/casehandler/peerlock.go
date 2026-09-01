package casehandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/go-redsync/redsync/v4"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
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
			Help:      "Total number of GetOrCreate calls that failed to acquire the per-peer serialization lock within the timeout. VOIP-1438: as of the Redis distributed peer lock, this counter conflates two distinct causes -- the in-process 5s timeout, and the Redis lock budget (1s) being exhausted or held by another replica (ErrTaken). Cross-reference case_redis_peer_lock_fail_total (Redis-caused failures) and case_redis_peer_lock_fail_open_total (Redis outage, not counted as a failure here) to disambiguate.",
		},
	)

	// promRedisPeerLockFailTotal counts acquirePeerLock failures caused by
	// the Redis distributed lock layer (redisPeerLockBudget exhausted, or
	// another replica already holding the lock via ErrTaken). Incremented
	// exactly once per failure, from acquirePeerLock's single error branch
	// -- tryAcquireRedisLock itself never increments this, to avoid double
	// counting.
	promRedisPeerLockFailTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "case_redis_peer_lock_fail_total",
			Help:      "Total number of GetOrCreate calls that failed to acquire the cross-replica Redis peer lock (budget exhausted or held by another replica)",
		},
	)

	// promRedisPeerLockFailOpenTotal counts occasions where a Redis
	// failure (connectivity, etc.) caused tryAcquireRedisLock to fall back
	// to in-process-only locking rather than blocking GetOrCreate. This is
	// NOT counted as a failure (GetOrCreate proceeds normally) -- it is an
	// early-warning signal that Redis is unreachable and cross-pod
	// protection is currently degraded to reliance on the MySQL unique
	// constraint backstop alone.
	promRedisPeerLockFailOpenTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "case_redis_peer_lock_fail_open_total",
			Help:      "Total number of GetOrCreate calls that proceeded without the cross-replica Redis peer lock because Redis was unreachable (fail-open; in-process lock and the DB unique constraint remain as backstops)",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promPeerLockMapSize,
		promDeadlockRetryTotal,
		promDeadlockExhaustedTotal,
		promPeerLockTimeoutTotal,
		promRedisPeerLockFailTotal,
		promRedisPeerLockFailOpenTotal,
	)
}

// VOIP-1438: Redis distributed peer lock tuning. redisPeerLockExpiry
// bounds how long a lock may be held before Redis auto-expires it (no
// auto-renewal -- redsyncMutex here intentionally omits ExtendContext; see
// main.go). GetOrCreate performs at most maxDeadlockRetries (3) full
// BeginTx-Commit cycles, so 10s gives wide margin over normal same-DC
// MySQL round-trip latency without leaving a crashed holder's lock stuck
// for too long.
const redisPeerLockExpiry = 10 * time.Second

// redisPeerLockTries/redisPeerLockRetryDelay configure redsync's internal
// bounded retry loop for LockContext. Deliberately independent of the
// in-process peerLockTimeout (5s): by the time tryAcquireRedisLock runs,
// the in-process lock has already absorbed same-pod contention, so the
// Redis layer only needs to wait out a peer being actively processed on a
// DIFFERENT replica.
const redisPeerLockTries = 5
const redisPeerLockRetryDelay = 200 * time.Millisecond

// redisPeerLockBudget is a hard, explicitly-enforced cap on
// tryAcquireRedisLock's total wall-clock time, applied via
// context.WithTimeout rather than relying solely on
// redisPeerLockTries*redisPeerLockRetryDelay. redsync v4.15.0's internal
// timeoutFactor (0.05) bounds each individual acquire attempt at
// redisPeerLockExpiry*0.05 (500ms here), so the tries*delay math alone
// could balloon to ~3.3s worst-case if Redis is alive but slow -- this
// budget forces a deterministic ~1s ceiling regardless.
const redisPeerLockBudget = 1 * time.Second

// redisPeerLockName namespaces the Redis lock key by service, reusing the
// same peerLockKey tuple the in-process lock uses so both layers guard
// identical critical sections.
func redisPeerLockName(key string) string {
	return "contact-manager:peerlock:" + key
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
// TOPOLOGY NOTE (VOIP-1438 supersedes the single-replica caveat that used
// to live here): this in-process lock alone is NOT cross-pod/distributed
// -- RabbitMQ's plain shared-queue competing-consumers model has no
// per-peer-tuple pod affinity, so under a multi-replica deployment two
// different replicas can concurrently process the same peer tuple. As of
// VOIP-1438, acquirePeerLock additionally acquires a Redis distributed
// lock (tryAcquireRedisLock, below) AFTER the in-process lock succeeds,
// closing that cross-pod gap. The two layers have non-overlapping
// responsibilities: this in-process lock serializes goroutines within a
// single pod (cheap, no Redis round-trip); the Redis lock serializes
// across pods. Both remain backstopped by insertWithRetry's
// uq_case_open_peer unique-constraint retry (getorcreate.go), which is
// the topology-independent correctness guarantee regardless of either
// lock's outcome.
func (h *caseHandler) acquirePeerLock(ctx context.Context, key string) (func(), error) {
	// 1) in-process lock (unchanged)
	ch := h.loadOrCreatePeerLockChan(key)

	lockCtx, cancel := context.WithTimeout(ctx, peerLockTimeout)
	defer cancel()

	select {
	case ch <- struct{}{}:
		// acquired
	case <-lockCtx.Done():
		return nil, ErrPeerLockTimeout
	}
	releaseLocal := func() { <-ch }

	// 2) Redis distributed lock (VOIP-1438) -- only attempted once the
	// in-process lock is held, avoiding an unnecessary Redis round-trip
	// when local contention alone already timed out.
	releaseRedis, err := h.tryAcquireRedisLock(ctx, key)
	if err != nil {
		releaseLocal()
		promRedisPeerLockFailTotal.Inc()
		return nil, fmt.Errorf("could not acquire distributed peer lock: %w", err)
	}

	return func() {
		releaseRedis()
		releaseLocal()
	}, nil
}

// tryAcquireRedisLock acquires the cross-replica Redis lock for key,
// bounded by redisPeerLockBudget. Returns a release func on success.
//
// h.redisLocker == nil is a deliberate fail-open path used by unit tests
// that construct caseHandler as a struct literal without redsync wiring
// (see main.go's caseHandler.redisLocker doc comment) -- production
// callers always go through NewCaseHandler, which never leaves
// redisLocker nil.
func (h *caseHandler) tryAcquireRedisLock(ctx context.Context, key string) (func(), error) {
	if h.redisLocker == nil {
		return func() {}, nil // fail-open: no locker wired (e.g. unit tests, contact-control CLI)
	}

	mutex := h.redisLocker.NewMutex(
		redisPeerLockName(key),
		redsync.WithExpiry(redisPeerLockExpiry),
		redsync.WithTries(redisPeerLockTries),
		redsync.WithRetryDelay(redisPeerLockRetryDelay),
	)

	// redsync's internal timeoutFactor alone cannot guarantee the ~1s
	// budget documented above (see redisPeerLockBudget's comment) -- this
	// context.WithTimeout enforces it explicitly.
	budgetCtx, cancel := context.WithTimeout(ctx, redisPeerLockBudget)
	defer cancel()

	err := mutex.LockContext(budgetCtx)
	if err == nil {
		return func() {
			if _, errUnlock := mutex.UnlockContext(ctx); errUnlock != nil {
				logrus.Warnf("could not release contact-manager peer lock %q. err: %v", key, errUnlock)
			}
		}, nil
	}

	// budgetCtx.Err() is checked BEFORE the RedisError type-assertion
	// below, and deliberately so: redsync v4.15.0's lockContext, when
	// budgetCtx expires mid-retry (e.g. during its final attempt), can
	// return the failure wrapped as &redsync.RedisError{} rather than a
	// plain context-cancellation error (actOnPoolsAsync wraps the
	// in-flight SetNX failure this way). If the RedisError branch below
	// ran first, that timing-dependent wrapping would misclassify a
	// budget-exhaustion (which must fail-closed, deterministically) as a
	// Redis-outage fail-open. Checking budgetCtx.Err() first keeps that
	// outcome deterministic regardless of which internal error shape
	// redsync happens to produce.
	if budgetCtx.Err() != nil {
		// Counter increment intentionally omitted here -- the caller
		// (acquirePeerLock) increments promRedisPeerLockFailTotal exactly
		// once for every non-nil error this function returns. Doing it
		// here too would double-count.
		return nil, fmt.Errorf("redis peer lock budget exhausted: %w", err)
	}

	var redisErr *redsync.RedisError
	if errors.As(err, &redisErr) {
		// Redis is unreachable (budgetCtx still live -- e.g. connection
		// refused immediately): fail-open. This is safe for a DIFFERENT
		// reason than bin-route-manager's healthcheck lock (which is safe
		// because its guarded write is a CAS-free blind UPDATE):
		// contact-manager's guarded write goes through insertWithRetry,
		// whose uq_case_open_peer unique constraint plus
		// re-query-and-use-the-winner logic already makes concurrent
		// INSERTs safe. A Redis outage without this fail-open would stall
		// GetOrCreate fleet-wide on every peer tuple; fail-open keeps
		// case creation working (with reduced cross-pod dedup, backstopped
		// by the unique constraint) at the cost of a possible uptick in
		// MySQL deadlock retries.
		logrus.Warnf("could not reach Redis for peer lock %q; proceeding with in-process lock only (fail-open). err: %v", key, err)
		promRedisPeerLockFailOpenTotal.Inc()
		return func() {}, nil
	}

	// Any other error (most commonly redsync.ErrTaken -- another replica
	// currently holds this peer tuple's lock, with budgetCtx still live)
	// fails closed: returning here would race with whichever replica
	// actually holds the lock. Counter increment is the caller's
	// responsibility, as above.
	return nil, err
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
