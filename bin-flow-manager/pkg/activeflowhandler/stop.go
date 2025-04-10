package activeflowhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/activeflow"
)

// Stop stops activeflow
func (h *activeflowHandler) Stop(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {

	af, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get activeflow info. activeflow_id: %s", id)
	}

	if af.Status == activeflow.StatusEnded {
		// already ended. nothing to do
		return af, nil
	}

	if errSet := h.db.ActiveflowSetStatus(ctx, id, activeflow.StatusEnded); errSet != nil {
		return nil, errors.Wrapf(errSet, "Could not set activeflow status. status: %s", activeflow.StatusEnded)
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated activeflow info. activeflow_id: %s", id)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveflowUpdated, res)

	return res, nil
}

// stopWithoutReturn stops the activeflow without returning the result.
func (h *activeflowHandler) stopWithoutReturn(ctx context.Context, id uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "stopWithoutReturn",
		"activeflow_id": id,
	})

	tmp, err := h.Stop(ctx, id)
	if err != nil {
		log.Errorf("could not stop the activeflow. activeflow_id: %s, err: %v", id, err)
		return
	}

	log.Debugf("stopped the activeflow. activeflow_id: %s", tmp.ID)
}
