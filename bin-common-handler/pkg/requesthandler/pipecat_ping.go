package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	outline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

const requestTimeoutPipecatPing = 1000 // pipecat ping timeout(1 sec)

// PipecatV1Ping issues a sub-second liveness probe against the per-pod queue
// for hostID. Returns nil if the pod responded with a matching host_id (the
// live case), or an error otherwise (timeout, circuit open, mismatched
// host_id from a queue-name collision, etc.).
//
// IMPORTANT: do not add status-code checks here. ANY response from the per-pod
// queue (200, 404 from old pods, even 5xx) means the pod is up and consuming
// — that is the only liveness signal we want. The only "dead" signal is
// err != nil, including ctx.DeadlineExceeded and circuitbreakerhandler.ErrCircuitOpen.
// Adding 5xx-as-error logic would defeat the purpose of the probe.
func (r *requestHandler) PipecatV1Ping(ctx context.Context, hostID string) error {
	queueName := fmt.Sprintf("%s.%s", outline.QueueNamePipecatRequest, hostID)
	res, err := r.sendRequest(
		ctx,
		outline.QueueName(queueName),
		"/v1/ping",
		sock.RequestMethodGet,
		"pipecat/ping",
		requestTimeoutPipecatPing,
		0,
		ContentTypeNone,
		nil,
	)
	if err != nil {
		return err
	}

	// Best-effort host_id echo verification. Note: Calico POD_IP recycle gives
	// a matching IP, so this check does NOT cover that case (see design §4.6).
	// For old pods the body is empty (404 simpleResponse) → skip the check.
	if res != nil && res.StatusCode == 200 && len(res.Data) > 0 {
		var pr pmpipecatcall.PingResult
		if errParse := json.Unmarshal(res.Data, &pr); errParse == nil {
			if pr.HostID != "" && pr.HostID != hostID {
				return fmt.Errorf("ping host_id mismatch: requested %s, got %s", hostID, pr.HostID)
			}
		}
	}
	return nil
}
