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

// ProcessTerminate starts the aicall terminating process.
func (h *aicallHandler) ProcessTerminate(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "ProcessTerminate",
		"aicall_id": id,
	})
	log.Debugf("Terminating the aicall. aicall_id: %s", id)

	// get aicall
	tmp, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the aicall correctly")
	}

	if tmp.Status == aicall.StatusTerminated {
		log.Debugf("Aicall is already terminated. aicall_id: %s", id)
		return tmp, nil
	}

	if errStop := h.reqHandler.FlowV1ActiveflowServiceStop(ctx, tmp.ActiveflowID, tmp.ID, 0); errStop != nil {
		log.Infof("Could not stop the service. But stopping the aicall anyways. err: %v", errStop)
	}

	if tmp.PipecatcallID != uuid.Nil {
		// terminate the pipecatcall
		pc, err := h.reqHandler.PipecatV1PipecatcallGet(ctx, tmp.PipecatcallID)
		if err != nil {
			return nil, errors.Wrap(err, "could not get the pipecatcall correctly")
		}

		log.WithField("pipecatcall", pc).Debugf("Terminating the pipecatcall. pipecatcall_id: %s", pc.ID)
		tmp, err := h.reqHandler.PipecatV1PipecatcallTerminate(ctx, pc.HostID, pc.ID)
		if err != nil {
			log.Errorf("Could not terminate the pipecatcall. err: %v", err)
		} else {
			log.WithField("pipecatcall", tmp).Debugf("Terminated the pipecatcall. pipecatcall_id: %s", tmp.ID)
		}
	}

	if tmp.ConfbridgeID != uuid.Nil {
		// terminate the confbridge
		cb, err := h.reqHandler.CallV1ConfbridgeTerminate(ctx, tmp.ConfbridgeID)
		if err != nil {
			// we could not terminate the confbridge, but we don't return the error here.
			// just log and continue
			log.Errorf("Could not terminate the confbridge. err: %v", err)
		} else {
			log.WithField("confbridge", cb).Debugf("Terminated the confbridge. confbridge_id: %s", cb.ID)
		}
	}

	res, err := h.UpdateStatus(ctx, id, aicall.StatusTerminated)
	if err != nil {
		return nil, errors.Wrap(err, "could not terminate the aicall")
	}

	return res, nil
}
