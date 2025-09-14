package aicallhandler

import (
	"context"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aicall"
)

// ProcessStart starts a aicall process
func (h *aicallHandler) ProcessStart(ctx context.Context, ac *aicall.AIcall) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ProcessStart",
		"aicall_id": ac.ID,
	})

	referenceType := tmtranscribe.ReferenceTypeCall

	// create transcribe
	tr, err := h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		ac.CustomerID,
		ac.ActiveflowID,
		uuid.Nil,
		referenceType,
		ac.ReferenceID,
		ac.Language,
		tmtranscribe.DirectionIn,
		30000,
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not start the transcribe")
	}
	log.WithField("transcribe", tr).Debugf("Started transcribe. transcribe_id: %s", tr.ID)

	// update status
	res, err := h.UpdateStatusStartProgressing(ctx, ac.ID, tr.ID)
	if err != nil {
		log.Errorf("Could not update the status to start. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ProcessPause pauses the aicall process
func (h *aicallHandler) ProcessPause(ctx context.Context, ac *aicall.AIcall) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ProcessPause",
		"aicall_id": ac.ID,
	})

	// stop the transcribe
	if ac.TranscribeID != uuid.Nil {
		_, err := h.reqHandler.TranscribeV1TranscribeStop(ctx, ac.TranscribeID)
		if err != nil {
			// failed to stop the transcribe but we keep move
			log.Errorf("Could not stop the transcribe. err: %v", err)
		}
	}

	res, err := h.UpdateStatusPausing(ctx, ac.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not end the aicall")
	}

	return res, nil
}

// ProcessTerminate ends a aicall process
func (h *aicallHandler) ProcessTerminate(ctx context.Context, ac *aicall.AIcall) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ProcessEnd",
		"aicall_id": ac.ID,
	})

	// stop the transcribe
	if ac.TranscribeID != uuid.Nil {
		_, err := h.reqHandler.TranscribeV1TranscribeStop(ctx, ac.TranscribeID)
		if err != nil {
			// failed to stop the transcribe but we keep move
			log.Errorf("Could not stop the transcribe. err: %v", err)
		}
	}

	// terminate the confbridge
	tmp, err := h.reqHandler.CallV1ConfbridgeTerminate(ctx, ac.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not terminate the confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not terminate the confbridge")
	}
	log.WithField("confbridge", tmp).Debugf("Terminated the confbridge. confbridge_id: %s", tmp.ID)

	res, err := h.UpdateStatusTerminated(ctx, ac.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not end the aicall")
	}

	return res, nil
}

// ProcessTerminating starts the aicall terminating process.
func (h *aicallHandler) ProcessTerminating(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	res, err := h.UpdateStatusFinishing(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "could not terminating the aicall")
	}

	return res, nil
}
