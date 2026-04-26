package aicallhandler

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
)

// pingPipecatHost runs a ~1s preflight against hostID. Returns true if the
// pod is reachable and owns this host_id; false if the pod is unreachable
// (timeout) or the breaker is open. Broker/transport errors return false
// and are logged distinctly so an outage is not misclassified as pod death.
//
// Note: relies on errors.Is unwrapping through pkg/errors v0.9.0+ wrappers
// applied by sendRequest. Verified pkg/errors v0.9.1 is pinned in
// bin-common-handler/go.mod and bin-ai-manager/go.mod.
func (h *aicallHandler) pingPipecatHost(ctx context.Context, hostID string) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":    "pingPipecatHost",
		"host_id": hostID,
	})
	if hostID == "" {
		return false
	}
	// PipecatV1Ping enforces its own 1s hard timeout inside sendRequest; this
	// outer 1.1s context provides a slightly wider cancellation safety net so
	// the helper still returns even if upstream contexts are uncancellable.
	cctx, cancel := context.WithTimeout(ctx, 1100*time.Millisecond)
	defer cancel()
	err := h.reqHandler.PipecatV1Ping(cctx, hostID)
	if err == nil {
		log.Debug("Pipecat host ping succeeded.")
		return true
	}
	switch {
	case errors.Is(err, circuitbreakerhandler.ErrCircuitOpen):
		log.Debug("Pipecat host ping skipped: circuit breaker open.")
	case errors.Is(err, context.DeadlineExceeded):
		log.Info("Pipecat host ping timed out; treating as dead.")
	default:
		log.Warnf("Pipecat ping failed with unexpected error; skipping per-pod RPC. err: %v", err)
	}
	return false
}
