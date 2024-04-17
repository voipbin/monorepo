package chatbotcallhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
)

// ProcessStart starts a chatbotcall process
func (h *chatbotcallHandler) ProcessStart(ctx context.Context, cb *chatbotcall.Chatbotcall) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ProcessStart",
		"chatbotcall_id": cb.ID,
	})

	referenceType := tmtranscribe.ReferenceTypeCall

	// create transcribe
	tr, err := h.reqHandler.TranscribeV1TranscribeStart(ctx, cb.CustomerID, referenceType, cb.ReferenceID, cb.Language, tmtranscribe.DirectionIn)
	if err != nil {
		log.Errorf("Could not create start the transcribe. err: %v", err)
		return nil, errors.Wrap(err, "could not start the transcribe")
	}
	log.WithField("transcribe", tr).Debugf("Started transcribe. transcribe_id: %s", tr.ID)

	// update status
	res, err := h.UpdateStatusStart(ctx, cb.ID, tr.ID)
	if err != nil {
		log.Errorf("Could not update the status to start. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ProcessEnd ends a chatbotcall process
func (h *chatbotcallHandler) ProcessEnd(ctx context.Context, cb *chatbotcall.Chatbotcall) (*chatbotcall.Chatbotcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ProcessEnd",
		"chatbotcall_id": cb.ID,
	})

	// stop the transcribe
	_, err := h.reqHandler.TranscribeV1TranscribeStop(ctx, cb.TranscribeID)
	if err != nil {
		// failed to stop the transcribe but we keep move
		log.Errorf("Could not stops the transcribe. err: %v", err)
	}

	res, err := h.UpdateStatusEnd(ctx, cb.ID)
	if err != nil {
		log.Errorf("Could not end the chatbotcall. err: %v", err)
		return nil, errors.Wrap(err, "could not end the chatbotcall")
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
