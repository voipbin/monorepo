package aicallhandler

import (
	"context"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aicall"
)

// ProcessStart starts a aicall process
func (h *aicallHandler) ProcessStart(ctx context.Context, cb *aicall.AIcall) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ProcessStart",
		"aicall_id": cb.ID,
	})

	referenceType := tmtranscribe.ReferenceTypeCall

	// create transcribe
	tr, err := h.reqHandler.TranscribeV1TranscribeStart(ctx, cb.CustomerID, referenceType, cb.ReferenceID, cb.Language, tmtranscribe.DirectionIn)
	if err != nil {
		return nil, errors.Wrap(err, "could not start the transcribe")
	}
	log.WithField("transcribe", tr).Debugf("Started transcribe. transcribe_id: %s", tr.ID)

	// update status
	res, err := h.UpdateStatusStartProgressing(ctx, cb.ID, tr.ID)
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
	_, err := h.reqHandler.TranscribeV1TranscribeStop(ctx, ac.TranscribeID)
	if err != nil {
		// failed to stop the transcribe but we keep move
		log.Errorf("Could not stops the transcribe. err: %v", err)
	}

	res, err := h.UpdateStatusPausing(ctx, ac.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not end the aicall")
	}

	return res, nil
}

// ProcessEnd ends a aicall process
func (h *aicallHandler) ProcessEnd(ctx context.Context, cb *aicall.AIcall) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ProcessEnd",
		"aicall_id": cb.ID,
	})

	// stop the transcribe
	_, err := h.reqHandler.TranscribeV1TranscribeStop(ctx, cb.TranscribeID)
	if err != nil {
		// failed to stop the transcribe but we keep move
		log.Errorf("Could not stops the transcribe. err: %v", err)
	}

	res, err := h.UpdateStatusEnd(ctx, cb.ID)
	if err != nil {
		return nil, errors.Wrap(err, "could not end the aicall")
	}

	// destroy the confbridge
	tmp, err := h.reqHandler.CallV1ConfbridgeDelete(ctx, cb.ConfbridgeID)
	if err != nil {
		// we couldn't delete the confbridge here.
		// but we don't return any error here because it doesn't affect to the activeflow execution.
		log.Errorf("Could not delete the confbridge. err: %v", err)
	}
	log.WithField("confbridge", tmp).Debugf("Destroyed the confbridge. confbridge_id: %s", tmp.ID)

	return res, nil
}
