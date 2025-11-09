package aicallhandler

import (
	"context"

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
	log.WithField("aicall", ac).Debug("Starting aicall process")

	// update status
	res, err := h.UpdateStatus(ctx, ac.ID, aicall.StatusProgressing)
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
	log.WithField("aicall", ac).Debug("Pausing aicall process")

	// stop pipecatcall
	if ac.PipecatcallID != uuid.Nil {
		// todo: need to fix.
		_, err := h.reqHandler.PipecatV1PipecatcallTerminate(ctx, "", ac.PipecatcallID)
		if err != nil {
			// failed to stop the pipecatcall but we keep move
			log.Errorf("Could not terminate the pipecatcall. err: %v", err)
		}
	}

	res, err := h.UpdateStatus(ctx, ac.ID, aicall.StatusPausing)
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
	log.Debugf("Terminating aicall process. aicall: %v", ac)

	// stop the pipecatcall
	if ac.PipecatcallID != uuid.Nil {
		// todo: need to fix
		log.Debugf("Terminating the pipecatcall. pipecatcall_id: %s", ac.PipecatcallID)
		tmp, err := h.reqHandler.PipecatV1PipecatcallTerminate(ctx, "", ac.PipecatcallID)
		if err != nil {
			// failed to stop the pipecatcall but we keep move
			log.Errorf("Could not terminate the pipecatcall. err: %v", err)
		} else {
			log.WithField("pipecatcall", tmp).Debugf("Terminated the pipecatcall. pipecatcall_id: %s", tmp.ID)
		}
	}

	// terminate the confbridge
	tmp, err := h.reqHandler.CallV1ConfbridgeTerminate(ctx, ac.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not terminate the confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not terminate the confbridge")
	}
	log.WithField("confbridge", tmp).Debugf("Terminated the confbridge. confbridge_id: %s", tmp.ID)

	res, err := h.UpdateStatus(ctx, ac.ID, aicall.StatusTerminated)
	if err != nil {
		return nil, errors.Wrap(err, "could not end the aicall")
	}

	return res, nil
}

// ProcessTerminating starts the aicall terminating process.
func (h *aicallHandler) ProcessTerminating(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	res, err := h.UpdateStatus(ctx, id, aicall.StatusTerminating)
	if err != nil {
		return nil, errors.Wrap(err, "could not terminating the aicall")
	}

	return res, nil
}
