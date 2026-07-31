package dispatchhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/models/schedule"
)

// dispatch fires the schedule's RPC with in-run retries and finalizes the
// execution row (design §5.5). Failure (transport error or non-2xx) is
// re-attempted up to retry_max times with linear backoff, all within the same
// execution row. The failed execution row IS the dead-letter record.
func (h *dispatchHandler) dispatch(ctx context.Context, s *schedule.Schedule, exec *execution.Execution) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "dispatch",
		"schedule_id":   s.ID,
		"schedule_name": s.Name,
		"execution_id":  exec.ID,
	})
	// the execution UUID is logged at dispatch start/end for correlation with
	// downstream logs (design §3: RequestID is not caller-settable)
	log.Debugf("Dispatching the schedule. target_queue: %s, target_uri: %s, target_method: %s", s.TargetQueue, s.TargetURI, s.TargetMethod)

	start := h.utilHandler.TimeNow()

	maxAttempts := s.RetryMax + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var res *sock.Response
	var errSend error
	attempts := 0
	success := false

attemptLoop:
	for i := 0; i < maxAttempts; i++ {
		attempts = i + 1

		res, errSend = h.reqHandler.SendRequest(
			ctx,
			commonoutline.QueueName(s.TargetQueue),
			s.TargetURI,
			sock.RequestMethod(s.TargetMethod),
			s.TimeoutMS,
			0,
			s.TargetDataType,
			s.TargetData,
		)
		if errSend == nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			success = true
			break
		}

		log.Errorf("Dispatch attempt failed. attempt: %d, err: %v", attempts, errSend)

		if i < maxAttempts-1 {
			select {
			case <-time.After(h.retryBackoff):
			case <-ctx.Done():
				break attemptLoop
			}
		}
	}

	end := h.utilHandler.TimeNow()
	durationMS := int(end.Sub(*start).Milliseconds())

	status := execution.StatusFailed
	statusCode := 0
	result := ""
	errStr := ""
	if success {
		status = execution.StatusSuccess
		statusCode = res.StatusCode
		result = truncateResult(string(res.Data))
	} else {
		if res != nil {
			statusCode = res.StatusCode
		}
		if errSend != nil {
			errStr = errSend.Error()
		} else {
			errStr = fmt.Sprintf("dispatch failed with status code %d", statusCode)
		}
	}

	completed, err := h.db.ExecutionComplete(ctx, exec.ID, status, statusCode, errStr, result, attempts, durationMS, *end)
	if err != nil {
		log.Errorf("Could not complete the execution. err: %v", err)
		return
	}
	if !completed {
		// the row was already finalized by the reaper (abandoned) — one
		// terminal state, no double-count in dispatch_total (design §5.6)
		log.Warnf("Execution was already finalized by the reaper. execution_id: %s", exec.ID)
		return
	}

	promDispatchTotal.WithLabelValues(s.Name, string(status)).Inc()
	promDispatchDurationMS.WithLabelValues(s.Name).Observe(float64(durationMS))
	if status == execution.StatusSuccess {
		promScheduleLastSuccessTimestamp.WithLabelValues(s.Name).Set(float64(end.Unix()))
	}

	h.notifyExecutionCompleted(ctx, exec.ID, status)

	log.Debugf("Dispatch completed. status: %s, status_code: %d, attempts: %d, duration_ms: %d", status, statusCode, attempts, durationMS)
}

// notifyExecutionCompleted re-fetches the finalized execution row and
// publishes the internal completion event (get-after-write, conventions §8.4).
func (h *dispatchHandler) notifyExecutionCompleted(ctx context.Context, id uuid.UUID, status execution.Status) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "notifyExecutionCompleted",
		"execution_id": id,
	})

	res, err := h.db.ExecutionGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the finalized execution info. err: %v", err)
		return
	}

	eventType := schedule.EventTypeExecutionFailed
	if status == execution.StatusSuccess {
		eventType = schedule.EventTypeExecutionSucceeded
	}

	h.notifyHandler.PublishEvent(ctx, eventType, res)
}

// truncateResult bounds the stored RPC response body to resultMaxBytes
// (design §4.2 — the column is an audit aid, not a parseable contract).
func truncateResult(s string) string {
	if len(s) <= resultMaxBytes {
		return s
	}

	return s[:resultMaxBytes]
}
