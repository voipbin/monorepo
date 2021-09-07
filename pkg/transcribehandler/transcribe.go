package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// TranscribeGet returns transcribe
func (h *transcribeHandler) TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "TranscribeGet",
		},
	)

	res, err := h.db.TranscribeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// TranscribeCreate create transcribe
func (h *transcribeHandler) TranscribeCreate(ctx context.Context, trans *transcribe.Transcribe) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "TranscribeUpdate",
		},
	)

	trans.TMCreate = getCurTime()
	trans.TMUpdate = defaultTimeStamp
	trans.TMDelete = defaultTimeStamp

	if err := h.db.TranscribeCreate(ctx, trans); err != nil {
		log.Errorf("Could not create transcribe. err: %v", err)
		return err
	}

	return nil
}
