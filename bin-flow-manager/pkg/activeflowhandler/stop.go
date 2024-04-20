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
	log := logrus.WithFields(logrus.Fields{
		"func":          "Stop",
		"activeflow_id": id,
	})

	af, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}

	if af.Status == activeflow.StatusEnded {
		// already ended. nothing to do
		return af, nil
	}

	if errSet := h.db.ActiveflowSetStatus(ctx, id, activeflow.StatusEnded); errSet != nil {
		log.Errorf("Could not set activeflow status. err: %v", errSet)
		return nil, errors.Wrap(errSet, "Could not set activeflow status.")
	}

	// get deleted activeflow
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, activeflow.EventTypeActiveflowUpdated, res)

	return res, nil
}
