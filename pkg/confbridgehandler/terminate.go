package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// Terminating starts terminating the conference
func (h *confbridgeHandler) Terminating(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Terminating",
		"confbridge_id": id,
	})
	log.Debug("Terminating the confbridge.")

	// get confbridge
	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return nil, errors.Wrap(err, "could not get confbridge info")
	}

	if tmp.Status == confbridge.StatusTerminated || tmp.Status == confbridge.StatusTerminating || tmp.TMDelete != dbhandler.DefaultTimeStamp {
		log.Infof("The confbridge is already terminated. status: %s", tmp.Status)
		return tmp, nil
	}

	// update the status to the terminating
	res, err := h.UpdateStatus(ctx, id, confbridge.StatusTerminating)
	if err != nil {
		log.Errorf("Could not update the status to terminating. err: %v", err)
		return nil, errors.Wrap(err, "could not update the status to terminating")
	}
	log.WithField("confbridge", res).Debugf("Updated confbridge status to terminating. confbridge_id: %s", res.ID)

	if res.BridgeID == "" {
		// no bridge allocated yet. just terminate the confbridge
		if errTerminate := h.Terminate(ctx, res.ID); errTerminate != nil {
			log.Errorf("Could not terminate the confbridge. err: %v", errTerminate)
			return nil, errors.Wrap(err, "could not terminate the confbridge")
		}
		return res, nil
	}

	// destroy the confbridge bridge
	if errDestroy := h.bridgeHandler.Destroy(ctx, res.BridgeID); errDestroy != nil {
		log.Errorf("Could not destroy confbridge bridge. err: %v", errDestroy)
		return nil, errors.Wrap(errDestroy, "could not destroy the confbridge bridge")
	}

	return res, nil
}

// Terminate terminates the conference
func (h *confbridgeHandler) Terminate(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Terminate",
		"confbridge_id": id,
	})
	log.Debug("Terminating the confbridge.")

	// get confbridge
	tmp, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return errors.Wrap(err, "could not get confbridge info")
	}

	if tmp.Status != confbridge.StatusTerminating {
		// the confbridge is not terminating
		return nil
	}

	// update the status to the terminating
	cb, err := h.UpdateStatus(ctx, id, confbridge.StatusTerminated)
	if err != nil {
		log.Errorf("Could not update the status to terminating. err: %v", err)
		return errors.Wrap(err, "could not update the status to terminating")
	}
	log.WithField("confbridge", cb).Debugf("Updated confbridge status to terminating. confbridge_id: %s", cb.ID)

	return nil
}
