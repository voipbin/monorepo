package pipecatcallhandler

import (
	"context"
	"time"

	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

// Ping returns this pod's identity for liveness verification by callers.
// Returning error keeps the door open for forward-compatible drain semantics
// (e.g., responding 503 during shutdown). v1 always returns nil.
func (h *pipecatcallHandler) Ping(ctx context.Context) (*pipecatcall.PingResult, error) {
	return &pipecatcall.PingResult{
		HostID:    h.hostID,
		Timestamp: time.Now().UTC(),
	}, nil
}
