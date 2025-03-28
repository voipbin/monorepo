package conferencehandler

import (
	"context"
	"fmt"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conference-manager/models/conference"
)

// TranscribeStart starts the conference transcribe.
func (h *conferenceHandler) TranscribeStart(ctx context.Context, id uuid.UUID, lang string) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeStart",
		"conference_id": id,
		"language":      lang,
	})

	tmp, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get conference info")
	}

	if tmp.Status != conference.StatusProgressing {
		return nil, errors.Wrapf(err, "invalid conference status")
	}

	tr, err := h.reqHandler.TranscribeV1TranscribeStart(ctx, tmp.CustomerID, uuid.Nil, uuid.Nil, tmtranscribe.ReferenceTypeConfbridge, tmp.ConfbridgeID, lang, tmtranscribe.DirectionIn, 30000)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the transcribe")
	}
	log.WithField("transcribe", tr).Debugf("Started transcribe. transcribe_id: %s", tr.ID)

	res, err := h.UpdateTranscribeID(ctx, id, tr.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update transcribe id")
	}
	log.WithField("conference", res).Debugf("Started transcribe. conference_id: %s, transcribe_id: %s", res.ID, tr.ID)

	return res, nil
}

// TranscribeStop stops the conference transcribe.
func (h *conferenceHandler) TranscribeStop(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "TranscribeStop",
		"conference_id": id,
	})

	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	if tmp.Status != conference.StatusProgressing {
		log.Errorf("Invalid conference status. status: %s", tmp.Status)
		return nil, fmt.Errorf("invalid conference status")
	} else if tmp.TranscribeID == uuid.Nil {
		log.Errorf("Conference has no ongoing transcribe. conference_id: %s", tmp.ID)
		return nil, fmt.Errorf("have no transcribe id")
	}

	tr, err := h.reqHandler.TranscribeV1TranscribeStop(ctx, tmp.TranscribeID)
	if err != nil {
		log.Errorf("Could not stop the transcribe. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", tr).Debugf("Stopped transcribe. transcribe_id: %s", tr.ID)

	res, err := h.UpdateTranscribeID(ctx, id, uuid.Nil)
	if err != nil {
		log.Errorf("Could not update transcribe id. err: %v", err)
		return nil, err
	}
	log.WithField("conference", res).Debugf("Stopped transcribe. conference_id: %s, transcribe_id: %s", res.ID, tr.ID)

	return res, nil
}
