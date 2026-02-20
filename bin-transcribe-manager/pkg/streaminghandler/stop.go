package streaminghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
)

// Stop stops the live streaming transcribe of the given streaming id
func (h *streamingHandler) Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Stop",
		"streaming_id": id,
	})

	st, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the streaming info. streaming_id: %s", id)
	}

	// note: the call-manager's external media id and streaming id are the same.
	em, err := h.reqHandler.CallV1ExternalMediaStop(ctx, st.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not stop the external media. external_media_id: %s", st.ID)
	}
	log.WithField("external_media", em).Debugf("Stopped external media. external_media_id: %s", em.ID)

	// Close the Asterisk WebSocket connection to release the file descriptor
	// and unblock the STT media processor's ReadMessage loop.
	if st.ConnAst != nil {
		_ = st.ConnAst.Close()
	}

	return st, nil
}
