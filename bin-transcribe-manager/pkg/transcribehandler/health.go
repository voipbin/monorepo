package transcribehandler

import (
	"context"
	stderrors "errors"

	"monorepo/bin-call-manager/models/call"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// HealthCheck checks the given transcribe is still vaild
// and stop the transcribe if the transcribe is not valid and over the default retry count.
func (h *transcribeHandler) HealthCheck(ctx context.Context, id uuid.UUID, retryCount int) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "HealthCheck",
		"transcribe_id": id,
		"retry_count":   retryCount,
	})

	// validate the transcribe. This guard must run unconditionally, before the
	// max-retry-exceeded branch below, so that a transcribe that is already
	// StatusDone or soft-deleted (TMDelete != nil) never enters
	// stopOrReschedule in the first place. TranscribeGet does not filter out
	// soft-deleted rows, so without this ordering a soft-deleted transcribe
	// with Status still "progressing" could be rescheduled by this health
	// check forever.
	tr, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return
	}
	if tr.Status == transcribe.StatusDone || tr.TMDelete != nil {
		// the transcribe is done or deleted already. no need to check the health anymore.
		return
	}

	if retryCount > defaultHealthMaxRetryCount {
		log.Errorf("The health check exceeded max retry count. Stopping the transcribe. retry_count: %d", retryCount)
		h.stopOrReschedule(ctx, tr, retryCount, log)
		return
	}

	// validate reference
	switch tr.ReferenceType {
	case transcribe.ReferenceTypeCall:
		c, err := h.reqHandler.CallV1CallGet(ctx, tr.ReferenceID)
		if err != nil {
			log.Errorf("Could not get reference call info. Stopping the transcribe. err: %v", err)
			h.stopOrReschedule(ctx, tr, retryCount, log)
			return
		}

		if c.Status == call.StatusHangup || c.TMDelete != nil || c.TMHangup != nil {
			// the call is done already. no need to check the health anymore.
			retryCount++
		} else {
			retryCount = 0
		}

	case transcribe.ReferenceTypeConfbridge:
		cb, err := h.reqHandler.CallV1ConfbridgeGet(ctx, tr.ReferenceID)
		if err != nil {
			log.Errorf("Could not get reference confbridge info. Stopping the transcribe. err: %v", err)
			h.stopOrReschedule(ctx, tr, retryCount, log)
			return
		}

		if cb.TMDelete != nil {
			retryCount++
		} else {
			retryCount = 0
		}
	}

	go func() {
		_ = h.reqHandler.TranscribeV1TranscribeHealthCheck(ctx, tr.HostID, id, defaultHealthDelay, retryCount)
	}()
}

// stopOrReschedule attempts to stop the transcribe. Stop() only fails when at
// least one streaming session could not genuinely be stopped (a
// cerrors.VoipbinError{Status: StatusFailedPrecondition, Reason:
// reasonStreamingStopFailed} - see pkg/transcribehandler/stop.go's
// zombie-session invariant) - in that case the transcribe is deliberately
// left in StatusProgressing rather than forced to StatusDone.
//
// IMPORTANT - this mechanism is currently dormant: as of this writing, nothing
// in the codebase schedules the *first* health check for a transcribe.
// `startLive` (pkg/transcribehandler/start.go) never calls
// TranscribeV1TranscribeHealthCheck, and no other service, cmd, or
// subscribehandler does either - the only callers of
// TranscribeV1TranscribeHealthCheck are this function's own reschedule call
// below and HealthCheck's own reschedule call. In other words, this reschedule
// loop only ever runs once some other change explicitly triggers the first
// health-check RPC for a transcribe; today it never fires on its own. Until
// that bootstrapping work happens (tracked separately - it is a distinct
// feature addition, not a safety fix, and is out of scope here), the only
// real recovery paths for a transcribe stuck in StatusProgressing after a
// genuine stop failure are (1) a manual POST /v1/transcribes/{id}/stop retry,
// or (2) Delete, which proceeds even if Stop fails (see transcribe.go's
// Delete). This function is still kept safe-by-construction below (narrow
// error match, bounded retries) so it is safe to wire up later without
// re-litigating this logic. Both TranscribeV1TranscribeStop and
// TranscribeV1TranscribeHealthCheck are now routed directly to the owner
// pod's per-pod queue (bin-manager.transcribe-manager-<host_id>.request)
// rather than the shared commonoutline.QueueNameTranscribeRequest queue, so
// Stop()'s isSafeToConsiderStopped (pkg/transcribehandler/stop.go) NotFound
// branch is safe regardless of replica count: a NotFound response always
// means that specific pod (host_id) has no session for this transcribe,
// which is a reliable "already stopped" signal independent of how many
// other replicas exist. See the detailed comment on isSafeToConsiderStopped
// for the full reasoning.
//
// If/when the first health check does get scheduled and this reschedule loop
// runs, ending it here on a genuine stop failure would leave that transcribe
// stuck in `progressing` forever with no remaining path to clean it up.
// Instead, reschedule another health check - but only:
//   - when the failure is specifically reasonStreamingStopFailed. Any other
//     error (e.g. an invalid reference type, a permanent DB failure, etc.)
//     cannot be fixed by retrying the exact same stop again, so we log and
//     give up instead of rescheduling forever.
//   - up to defaultStopRescheduleMaxRetryCount attempts, so a persistent
//     failure (e.g. call-manager unreachable indefinitely) eventually stops
//     rescheduling instead of looping forever.
func (h *transcribeHandler) stopOrReschedule(ctx context.Context, tr *transcribe.Transcribe, retryCount int, log *logrus.Entry) {
	_, err := h.Stop(ctx, tr.ID)
	if err == nil {
		return
	}

	var ve *cerrors.VoipbinError
	isRetryable := stderrors.As(err, &ve) && ve.Status == cerrors.StatusFailedPrecondition && ve.Reason == reasonStreamingStopFailed
	if !isRetryable {
		// a permanent failure: retrying the same stop again will never
		// succeed, so do not reschedule.
		log.Errorf("Could not stop the transcribe with a non-retryable error. Giving up without rescheduling. transcribe_id: %s, err: %v", tr.ID, err)
		return
	}

	if retryCount >= defaultStopRescheduleMaxRetryCount {
		log.Errorf("Could not stop the transcribe after %d attempts. Giving up; the transcribe remains in progressing and needs manual intervention (see docs/operations.md). transcribe_id: %s, err: %v", retryCount, tr.ID, err)
		return
	}

	log.Errorf("Could not stop the transcribe. Will retry via a future health check. transcribe_id: %s, err: %v", tr.ID, err)
	go func() {
		_ = h.reqHandler.TranscribeV1TranscribeHealthCheck(ctx, tr.HostID, tr.ID, defaultHealthDelay, retryCount+1)
	}()
}
