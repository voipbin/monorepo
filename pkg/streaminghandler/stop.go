package streaminghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
)

// Start starts the live streaming transcribe of the given transcribe
func (h *streamingHandler) Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Stop",
			"streaming_id": id,
		},
	)

	st, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get streaming info. err: %v", err)
		return nil, err
	}

	em, err := h.reqHandler.CallV1ExternalMediaStop(ctx, st.ExternalMediaID)
	if err != nil {
		log.Errorf("Could not stop the external media.")
		return nil, err
	}
	log.WithField("external_media", em).Debugf("Stopped external media. external_media_id: %s", em.ID)

	return st, nil
}
