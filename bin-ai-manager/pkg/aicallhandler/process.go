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
		pc, err := h.reqHandler.PipecatV1PipecatcallGet(ctx, ac.PipecatcallID)
		if err != nil {
			return nil, errors.Wrap(err, "could not get the pipecatcall correctly")
		}

		tmp, err := h.reqHandler.PipecatV1PipecatcallTerminate(ctx, pc.HostID, ac.PipecatcallID)
		if err != nil {
			log.Errorf("Could not terminate the pipecatcall. err: %v", err)
		} else {
			log.WithField("pipecatcall", tmp).Debugf("Terminated the pipecatcall. pipecatcall_id: %s", tmp.ID)
		}
	}

	res, err := h.UpdateStatus(ctx, ac.ID, aicall.StatusPausing)
	if err != nil {
		return nil, errors.Wrap(err, "could not end the aicall")
	}

	return res, nil
}

// ProcessTerminate ends a aicall process
func (h *aicallHandler) ProcessTerminate(ctx context.Context, c *aicall.AIcall) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ProcessTerminate",
		"aicall_id": c.ID,
	})
	log.Debugf("Terminating aicall process. aicall: %v", c)

	if c.Status == aicall.StatusTerminated {
		log.Debugf("Aicall is already terminated. aicall_id: %s", c.ID)
		return c, nil
	}

	// stop the pipecatcall
	if c.PipecatcallID != uuid.Nil {
		pc, err := h.reqHandler.PipecatV1PipecatcallGet(ctx, c.PipecatcallID)
		if err != nil {
			return nil, errors.Wrap(err, "could not get the pipecatcall correctly")
		}

		log.Debugf("Terminating the pipecatcall. pipecatcall_id: %s", c.PipecatcallID)
		tmp, err := h.reqHandler.PipecatV1PipecatcallTerminate(ctx, pc.HostID, c.PipecatcallID)
		if err != nil {
			log.Errorf("Could not terminate the pipecatcall. err: %v", err)
		} else {
			log.WithField("pipecatcall", tmp).Debugf("Terminated the pipecatcall. pipecatcall_id: %s", tmp.ID)
		}
	}

	// terminate the confbridge
	tmp, err := h.reqHandler.CallV1ConfbridgeTerminate(ctx, c.ConfbridgeID)
	if err != nil {
		// we could not terminate the confbridge, but we don't return the error here.
		// just log and continue
		log.Errorf("Could not terminate the confbridge. err: %v", err)
	} else {
		log.WithField("confbridge", tmp).Debugf("Terminated the confbridge. confbridge_id: %s", tmp.ID)
	}

	res, err := h.UpdateStatus(ctx, c.ID, aicall.StatusTerminated)
	if err != nil {
		return nil, errors.Wrap(err, "could not end the aicall")
	}

	return res, nil
}

// ProcessTerminating starts the aicall terminating process.
func (h *aicallHandler) ProcessTerminating(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ProcessTerminating",
		"aicall_id": id,
	})
	log.Debugf("Terminating the aicall. aicall_id: %s", id)

	res, err := h.UpdateStatus(ctx, id, aicall.StatusTerminating)
	if err != nil {
		return nil, errors.Wrap(err, "could not start terminating the aicall")
	}

	if errStop := h.reqHandler.FlowV1ActiveflowServiceStop(ctx, res.ActiveflowID, res.ID, 0); errStop != nil {
		return nil, errors.Wrapf(errStop, "could not stop the aicall")
	}

	// exit the call from the conference bridge
	if errKick := h.reqHandler.CallV1ConfbridgeCallKick(ctx, res.ConfbridgeID, res.ReferenceID); errKick != nil {
		return nil, errors.Wrapf(errKick, "could not kick the call from the conference bridge")
	}

	return res, nil
}
