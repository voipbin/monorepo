package streaminghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/models/streaming"
)

// Stop stops the live streaming transcribe of the given streaming id
func (h *streamingHandler) Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Stop",
		"streaming_id": id,
	})

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the streaming info. streaming_id: %s", id)
	}

	// note:
	// the call-manager's external media id and streaming id are the same.
	em, err := h.requestHandler.CallV1ExternalMediaStop(ctx, res.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not stop the external media. external_media_id: %s", res.ID)
	}
	log.WithField("external_media", em).Debugf("Stopped external media. external_media_id: %s", em.ID)

	return res, nil
}
