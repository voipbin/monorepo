package transcribehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// Stop stops the progressing transcribe process.
func (h *transcribeHandler) Stop(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Stop",
		"transcribe_id": id,
	})

	// get transcribe and evaluate
	tr, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe. err: %v", err)
		return nil, err
	}

	if tr.Status == transcribe.StatusDone {
		// already stopped
		log.WithField("transcribe", tr).Debugf("Already stopped. transcribe_id: %s", tr.ID)
		return tr, nil
	}

	switch tr.ReferenceType {
	case transcribe.ReferenceTypeCall, transcribe.ReferenceTypeConfbridge:
		return h.stopLive(ctx, tr)

	default:
		log.Errorf("Invalid reference type. reference_type: %s", tr.ReferenceType)
		return nil, fmt.Errorf("invalid reference type")
	}
}

// stopLive stops live transcribing.
func (h *transcribeHandler) stopLive(ctx context.Context, tr *transcribe.Transcribe) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "stopLive",
		"transcribe_id": tr.ID,
	})

	for _, streamingID := range tr.StreamingIDs {
		st, err := h.streamingHandler.Stop(ctx, streamingID)
		if err != nil {
			log.Errorf("Could not stop the streaming. streaming_id: %s, err: %v", streamingID, err)
			continue
		}
		log.WithField("streaming", st).Debugf("Stopped streaming. streaming_id: %s", st.ID)
	}

	res, err := h.UpdateStatus(ctx, tr.ID, transcribe.StatusDone)
	if err != nil {
		log.Errorf("Could not update the status. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", res).Debugf("Updated transcribe status done. transcribe_id: %s", res.ID)

	return res, nil
}
