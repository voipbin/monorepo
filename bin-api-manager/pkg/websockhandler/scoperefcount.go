package websockhandler

import (
	"sync"

	"monorepo/bin-common-handler/pkg/sockhandler"
)

// scopeRefCount tracks, per api-manager pod, how many local websocket connections currently
// have at least one active subscription for each AMQP binding pattern (e.g.
// "customer_id.<uuid>.#"), and binds/unbinds the pod's per-pod queue accordingly. This is the
// component that makes the fanout->topic conversion actually reduce per-pod processing: a scope
// with zero local subscribers is never bound, so the broker never delivers it to this pod.
// See VOIP-1258 design doc §9.
type scopeRefCount struct {
	mu       sync.Mutex
	counts   map[string]int // binding pattern -> active subscriber count
	sock     sockhandler.SockHandler
	queue    string
	exchange string
}

func newScopeRefCount(sock sockhandler.SockHandler, queueName string, exchangeName string) *scopeRefCount {
	return &scopeRefCount{
		counts:   make(map[string]int),
		sock:     sock,
		queue:    queueName,
		exchange: exchangeName,
	}
}

// Acquire increments the refcount for the given binding pattern, binding the queue on the
// first (0->1) transition.
func (r *scopeRefCount) Acquire(pattern string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.counts[pattern] == 0 {
		if err := r.sock.QueueBind(r.queue, pattern, r.exchange, false, nil); err != nil {
			return err
		}
	}
	r.counts[pattern]++
	return nil
}

// Release decrements the refcount for the given binding pattern, unbinding the queue on the
// last (1->0) transition. No-op (not an error) if the pattern isn't currently tracked, to
// tolerate double-release from racing cleanup paths (explicit unsubscribe + abrupt disconnect).
func (r *scopeRefCount) Release(pattern string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.counts[pattern] <= 0 {
		return nil
	}
	r.counts[pattern]--
	if r.counts[pattern] == 0 {
		delete(r.counts, pattern)
		if err := r.sock.QueueUnbind(r.queue, pattern, r.exchange, nil); err != nil {
			return err
		}
	}
	return nil
}

// ReleaseAll releases every currently-held pattern for this connection's tracked set. Used on
// abrupt disconnect (VOIP-1258 §9) where no per-topic unsubscribe message was received. Callers
// must pass the SET of patterns this specific connection held (tracked separately per
// connection, not by scopeRefCount itself -- see Task 4.3).
func (r *scopeRefCount) ReleaseAll(patterns []string) {
	for _, p := range patterns {
		_ = r.Release(p) // best-effort; log at call site if needed
	}
}
