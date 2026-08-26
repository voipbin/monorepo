package streaminghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// ErrSTTNotConfiguredReason is the VoipbinError reason returned by the disabled streaming
// handler's per-request methods when no STT provider (GCP or AWS) could be initialized at
// startup. The service keeps running so that every other transcribe-manager capability
// stays available; only the live-streaming transcribe path is unavailable. Matches this
// codebase's structured-error convention (see pkg/transcribehandler/start.go's
// TRANSCRIBE_ALREADY_PROGRESSING for another example) so API callers get a typed,
// domain/reason-tagged error instead of an opaque 500.
//
// Exported so that other packages can match on this specific reason instead of on
// Status alone. In particular, pkg/transcribehandler/stop.go's isSafeToConsiderStopped
// checks Status == StatusUnavailable && Reason == ErrSTTNotConfiguredReason before
// treating a streamingHandler.Stop failure as "safe to consider already stopped" -
// StatusUnavailable by itself is a general-purpose "transient, retry later" status
// that other call paths (e.g. a future typed-error migration of call-manager's
// CallV1ExternalMediaStop) could also return for genuinely transient failures, so the
// Reason match is required to avoid misclassifying a live session as already gone.
const ErrSTTNotConfiguredReason = "STT_NOT_CONFIGURED"

// newErrSTTNotConfigured returns a fresh VoipbinError for the disabled handler's per-request
// methods to return.
func newErrSTTNotConfigured() error {
	return cerrors.Unavailable(
		commonoutline.ServiceNameTranscribeManager,
		ErrSTTNotConfiguredReason,
		"No STT provider (GCP or AWS) is configured on this instance. Live-streaming transcribe is unavailable; all other transcribe-manager functionality is unaffected.",
	)
}

// disabledStreamingHandler is the StreamingHandler used when neither the GCP nor the AWS STT
// client could be initialized. It exists so that NewStreamingHandler never returns a nil
// interface, which used to make the service exit fatally at boot.
type disabledStreamingHandler struct{}

// NewDisabledStreamingHandler creates a StreamingHandler that reports STT as not configured
// on every per-request call, instead of failing at startup.
func NewDisabledStreamingHandler() StreamingHandler {
	return &disabledStreamingHandler{}
}

// Run is a no-op, matching the real streamingHandler.Run(). Returning an error here would
// abort service startup, which is exactly what this implementation exists to avoid.
func (h *disabledStreamingHandler) Run() error {
	logrus.WithField("func", "Run").Warn("Streaming transcribe is disabled: no STT provider is configured. All other transcribe-manager functions remain available.")
	return nil
}

// Start always fails, reporting STT as unavailable via a structured VoipbinError.
func (h *disabledStreamingHandler) Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction, provider transcribe.Provider) (*streaming.Streaming, error) {
	logrus.WithFields(logrus.Fields{
		"func":          "Start",
		"transcribe_id": transcribeID,
	}).Error("Could not start the streaming. No STT provider is configured.")

	return nil, newErrSTTNotConfigured()
}

// Stop always fails, reporting STT as unavailable via a structured VoipbinError.
func (h *disabledStreamingHandler) Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	logrus.WithFields(logrus.Fields{
		"func": "Stop",
		"id":   id,
	}).Error("Could not stop the streaming. No STT provider is configured.")

	return nil, newErrSTTNotConfigured()
}
