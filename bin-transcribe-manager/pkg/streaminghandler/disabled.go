package streaminghandler

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// ErrSTTNotConfigured is returned by the disabled streaming handler's per-request methods
// when no STT provider (GCP or AWS) could be initialized at startup. The service keeps
// running so that every other transcribe-manager capability stays available; only the
// live-streaming transcribe path is unavailable.
var ErrSTTNotConfigured = errors.New("STT_NOT_CONFIGURED: no STT provider available")

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

// Start always fails with ErrSTTNotConfigured.
func (h *disabledStreamingHandler) Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction, provider transcribe.Provider) (*streaming.Streaming, error) {
	logrus.WithFields(logrus.Fields{
		"func":          "Start",
		"transcribe_id": transcribeID,
	}).Error("Could not start the streaming. No STT provider is configured.")

	return nil, ErrSTTNotConfigured
}

// Stop always fails with ErrSTTNotConfigured.
func (h *disabledStreamingHandler) Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	logrus.WithFields(logrus.Fields{
		"func": "Stop",
		"id":   id,
	}).Error("Could not stop the streaming. No STT provider is configured.")

	return nil, ErrSTTNotConfigured
}
