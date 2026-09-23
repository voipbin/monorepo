package queuecallhandler

import (
	"context"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// List of default values for the matching backstop (VOIP-1539 §5.2).
const (
	// reconcileLockTTL bounds how long one replica's reconcile lease is
	// held -- comfortably above reconcileInterval so a slow pass never
	// races the next tick's SetNX, but short enough that a crashed
	// replica's lease expires and lets another replica pick up the next
	// tick.
	reconcileLockTTL = 35 * time.Second

	// connectingStaleAfter is the design's Tconn: how long a queuecall can
	// sit in the connecting status before the backstop treats it as
	// abandoned by the agent-dial path and rolls it back to waiting.
	// Deliberately generous relative to call-manager's own dial timeout
	// (design §5.2: "Tconn > call-manager 최대 agent-dial 타임아웃 + 마진") --
	// this is a backstop for a connecting queuecall that never resolved at
	// all (crashed dial, lost webhook), not a fast-path timeout.
	connectingStaleAfter = 60 * time.Second

	// reconcileScanLimit bounds how many rows each recovery path examines
	// per tick, so one pass can never issue an unbounded number of RPCs.
	reconcileScanLimit = 100
)

// reconcileInstanceID identifies this replica in the reconcile lease value.
// Best-effort only -- the lease's correctness never depends on the value,
// only on SetNX's atomicity; this just makes the current holder visible for
// debugging via `redis-cli GET queue:reconcile:match`.
var reconcileInstanceID = func() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "unknown"
	}
	return host
}()

// Reconcile runs one pass of the matching backstop (VOIP-1539 §5.2): a
// low-frequency active sweep that recovers from missed/lost event-driven
// match() triggers (§3.5) -- e.g. a subscribehandler restart window, a lost
// RabbitMQ delivery, or a dial that crashed mid-flight without resolving its
// entry CAS. Callers are expected to invoke this on a fixed interval (e.g.
// every 30s) from every replica; the Redis lease below ensures only one
// replica's pass actually runs a given tick.
func (h *queuecallHandler) Reconcile(ctx context.Context) {
	log := logrus.WithField("func", "Reconcile")

	acquired, err := h.cache.ReconcileLockAcquire(ctx, reconcileInstanceID, reconcileLockTTL)
	if err != nil {
		log.Errorf("Could not acquire the reconcile lease. err: %v", err)
		return
	}
	if !acquired {
		// Another replica already holds this tick's lease -- skip silently
		// (design §5.2 step 1).
		return
	}
	defer func() {
		if errRelease := h.cache.ReconcileLockRelease(ctx); errRelease != nil {
			log.Debugf("Could not release the reconcile lease. err: %v", errRelease)
		}
	}()

	h.reconcileConnectingStale(ctx)
	h.reconcileWaiting(ctx)
}

// reconcileConnectingStale is recovery A (design §5.2 step 2): queuecalls
// stuck in the connecting status past connectingStaleAfter get their
// groupcall hung up (if any) and rolled back to waiting via
// UpdateStatusWaitingRollback, which is itself CAS-gated and therefore
// idempotent against a queuecall that already progressed past connecting on
// its own (join CAS win) between the list and the rollback call.
func (h *queuecallHandler) reconcileConnectingStale(ctx context.Context) {
	log := logrus.WithField("func", "reconcileConnectingStale")

	cutoff := *h.utilHandler.TimeNowAdd(-connectingStaleAfter)

	qcs, err := h.db.QueuecallListConnectingStale(ctx, cutoff, reconcileScanLimit)
	if err != nil {
		log.Errorf("Could not list connecting-stale queuecalls. err: %v", err)
		return
	}

	for _, qc := range qcs {
		if qc.GroupcallID != uuid.Nil {
			if _, errHangup := h.reqHandler.CallV1GroupcallHangup(ctx, qc.GroupcallID); errHangup != nil {
				// The groupcall may already be gone (agent answered and
				// hung up on their own, or a prior backstop pass already
				// hung it up) -- log and continue the rollback regardless,
				// UpdateStatusWaitingRollback's own CAS is the real safety
				// net here.
				log.Debugf("Could not hang up the stale groupcall. queuecall_id: %s, groupcall_id: %s, err: %v", qc.ID, qc.GroupcallID, errHangup)
			}
		}

		if _, errRollback := h.UpdateStatusWaitingRollback(ctx, qc); errRollback != nil {
			log.Errorf("Could not roll back the stale connecting queuecall. queuecall_id: %s, err: %v", qc.ID, errRollback)
		}
	}
}

// reconcileWaiting is recovery B (design §5.2 step 3): every waiting
// queuecall gets a fresh, idempotent match() attempt via
// matchWaitingQueuecall -- the exact same one-shot entry-point-A logic, just
// re-triggered on a timer instead of enqueue. A queuecall with no available
// agent is a no-op (matchWaitingQueuecall's own early return); a queuecall
// that another entry point already matched in the meantime loses the
// reservation/entry CAS inside Execute and is likewise a no-op.
func (h *queuecallHandler) reconcileWaiting(ctx context.Context) {
	log := logrus.WithField("func", "reconcileWaiting")

	qcs, err := h.db.QueuecallListWaitingOldest(ctx, reconcileScanLimit)
	if err != nil {
		log.Errorf("Could not list waiting queuecalls. err: %v", err)
		return
	}

	for _, qc := range qcs {
		h.matchWaitingQueuecall(ctx, qc)
	}
}
