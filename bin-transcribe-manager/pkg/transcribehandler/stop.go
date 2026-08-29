package transcribehandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
)

// reasonStreamingStopFailed is the VoipbinError reason stopLive uses when at
// least one streaming session could not be confirmed stopped (the
// zombie-session invariant below). health.go's stopOrReschedule matches on
// this exact reason to decide whether a Stop() failure is worth retrying via
// another scheduled health check, as opposed to a permanent failure (e.g. an
// invalid reference type) that retrying can never fix.
const reasonStreamingStopFailed = "STREAMING_STOP_FAILED"

// Stop stops the progressing transcribe process.
func (h *transcribeHandler) Stop(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Stop",
		"transcribe_id": id,
	})

	// get transcribe and evaluate
	tr, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe. transcribe_id: %s", id)
	}
	log.WithField("transcribe", tr).Debugf("Found the transcribe. transcribe_id: %s", tr.ID)

	if tr.Status == transcribe.StatusDone {
		// already stopped
		log.WithField("transcribe", tr).Debugf("Already stopped. transcribe_id: %s", tr.ID)
		return tr, nil
	}

	var res *transcribe.Transcribe
	switch tr.ReferenceType {
	case transcribe.ReferenceTypeCall, transcribe.ReferenceTypeConfbridge:
		res, err = h.stopLive(ctx, tr)

	default:
		log.Errorf("Invalid reference type. reference_type: %s", tr.ReferenceType)
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameTranscribeManager,
			"INVALID_REFERENCE_TYPE",
			fmt.Sprintf("invalid reference type: %s", tr.ReferenceType),
		)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "could not stop the transcribe. transcribe_id: %s", tr.ID)
	}

	if res.OnEndFlowID == uuid.Nil {
		return res, nil
	}

	// create activeflow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, uuid.Nil, res.CustomerID, res.OnEndFlowID, fmactiveflow.ReferenceTypeTranscribe, res.ID, res.ActiveflowID, nil, "", fmactiveflow.WebhookMethodNone)
	if err != nil {
		// we could not create the activeflow, but continue to stop the transcribe
		log.Errorf("Could not create the activeflow. err: %v", err)
		return res, nil
	}
	log.WithField("activeflow", af).Debugf("Created activeflow. activeflow_id: %s", af.ID)

	if errSet := h.variableSet(ctx, af.ID, res); errSet != nil {
		// we could not set the variable, but continue to handle the on end flow execution
		log.Errorf("Could not set the variable. err: %v", errSet)
	}

	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); errExecute != nil {
		// we could not execute the activeflow, but continue to stop the transcribe
		log.Errorf("Could not execute the activeflow. activeflow_id: %s, err: %v", af.ID, errExecute)
		return res, nil
	}

	return res, nil
}

// stopLive stops live transcribing.
func (h *transcribeHandler) stopLive(ctx context.Context, tr *transcribe.Transcribe) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "stopLive",
		"transcribe_id": tr.ID,
	})

	var failedStreamingIDs []uuid.UUID
	for _, streamingID := range tr.StreamingIDs {
		st, err := h.streamingHandler.Stop(ctx, streamingID)
		if err != nil {
			if isSafeToConsiderStopped(err) {
				// the streaming session no longer exists (e.g., it already finished and was
				// cleaned up, or the external media was already stopped elsewhere), or it
				// cannot logically be alive (e.g., no STT provider is configured on this
				// instance, so nothing could ever have started it). Either way there is
				// nothing left to stop here. Safe to consider it already stopped and
				// continue with the other streamings.
				log.Infof("Could not stop the streaming. But consider already stopped. streaming_id: %s, err: %v", streamingID, err)
				continue
			}

			// a genuine stop failure: the streaming session may still be running. Do not
			// continue to mark the transcribe done for this, or the session becomes a
			// zombie that nothing can ever stop again. Keep trying the remaining
			// streamings so we stop as many as we can, but track the failure.
			log.Errorf("Could not stop the streaming. streaming_id: %s, err: %v", streamingID, err)
			failedStreamingIDs = append(failedStreamingIDs, streamingID)
			continue
		}
		log.WithField("streaming", st).Debugf("Stopped streaming. streaming_id: %s", st.ID)
	}

	if len(failedStreamingIDs) > 0 {
		return nil, cerrors.FailedPrecondition(
			commonoutline.ServiceNameTranscribeManager,
			reasonStreamingStopFailed,
			fmt.Sprintf("could not stop %d of %d streaming session(s). transcribe_id: %s, failed streaming_ids: %v", len(failedStreamingIDs), len(tr.StreamingIDs), tr.ID, failedStreamingIDs),
		)
	}

	res, err := h.UpdateStatus(ctx, tr.ID, transcribe.StatusDone)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the status. transcribe_id: %s", tr.ID)
	}
	log.WithField("transcribe", res).Debugf("Updated transcribe status done. transcribe_id: %s", res.ID)

	return res, nil
}

// isSafeToConsiderStopped decides whether an error returned from
// streamingHandler.Stop means the streaming session cannot possibly still be
// alive, so it is safe to treat the stop as having already happened and
// proceed to mark the transcribe done.
//
// This must recognize every error shape streamingHandler.Stop can actually
// produce, not just the typed one:
//
//   - typed cerrors.VoipbinError{Status: StatusNotFound}: the in-memory
//     streaming.Get() lookup missed (see streaminghandler/streaming.go's Get),
//     i.e. the session was already cleaned up locally.
//
//     This branch is safe to treat as "session cannot possibly be alive"
//     regardless of replica count. The in-memory session map (mapStreaming)
//     is per-pod, and a genuinely live session only exists in the memory of
//     the one pod that owns it - but both TranscribeV1TranscribeStop and
//     TranscribeV1TranscribeHealthCheck are now routed directly to that
//     owning pod's per-pod queue (bin-manager.transcribe-manager-<hostID>.
//     request, wired up in cmd/transcribe-manager/main.go) rather than the
//     shared commonoutline.QueueNameTranscribeRequest queue. Since the RPC
//     can only ever reach the pod identified by the transcribe's HostID, a
//     NotFound here always means "no session on the pod that owns this
//     transcribe" - which is the only pod that could ever have had one -
//     independent of how many other replicas exist.
//   - legacy requesthandler.ErrNotFound sentinel: the call-manager RPC
//     (CallV1ExternalMediaStop) surfaces a 404 through the older,
//     pre-VoipbinError sentinel path instead of a typed error (call-manager's
//     external-media error responses are not yet migrated to typed errors).
//     Mirrors the same dual-check pattern used by
//     bin-call-manager/pkg/channelhandler/hangup.go's HangingUpWithAsteriskID.
//   - typed cerrors.VoipbinError{Status: StatusUnavailable, Reason:
//     streaminghandler.ErrSTTNotConfiguredReason}: returned by
//     streaminghandler's disabledStreamingHandler.Stop (see
//     pkg/streaminghandler/disabled.go) when no STT provider is configured on
//     this instance. Start always fails in that state too, so no live session
//     could ever have existed to begin with — treating this as a genuine
//     failure would permanently block the StatusDone transition for an
//     instance that structurally can never have a real session to stop.
//
//     IMPORTANT: this branch intentionally checks Status AND Reason together,
//     unlike the NotFound branch above (which is a genuine dual-check mirror
//     of hangup.go's pattern and only needs Status). StatusUnavailable by
//     itself is a general-purpose "transient failure, safe to retry later"
//     status - if bin-call-manager's ARI/Asterisk error responses are ever
//     migrated to typed errors, a genuinely-alive session could plausibly
//     surface StatusUnavailable too (e.g. Asterisk temporarily unreachable).
//     Matching on Status alone would then misclassify a live session as
//     already stopped and reintroduce the exact zombie-session bug this file
//     exists to prevent. Narrowing to the specific STT_NOT_CONFIGURED reason
//     keeps this branch scoped to the one situation where "no session could
//     possibly exist" is actually true. This Reason-scoped check is a local
//     extension of the dual-check pattern, not something hangup.go's pattern
//     itself does.
func isSafeToConsiderStopped(err error) bool {
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		if ve.Status == cerrors.StatusNotFound {
			return true
		}
		if ve.Status == cerrors.StatusUnavailable && ve.Reason == streaminghandler.ErrSTTNotConfiguredReason {
			return true
		}
	}

	if stderrors.Is(err, requesthandler.ErrNotFound) {
		return true
	}

	return false
}
