package activeflowhandler

import (
	"context"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Delete deletes activeflow
func (h *activeflowHandler) Delete(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Delete",
		"activeflow_id": id,
	})

	// get activeflow
	a, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	// check the activeflow has been
	if a.TMDelete != dbhandler.DefaultTimeStamp {
		// already deleted
		return a, nil
	}

	if a.Status != activeflow.StatusEnded {
		log.Debugf("The activeflow is not ended. Stopping the activeflow. activeflow_id: %s, status: %s", a.Identity.ID, a.Status)
		tmp, err := h.Stop(ctx, id)
		if err != nil {
			log.Errorf("Could not stop the activeflow. err: %v", err)
			return nil, err
		}
		log.Debugf("Stopped activeflow. activeflow_id: %s", tmp.Identity.ID)
	}

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the activeflow. err: %v", err)
		return nil, err
	}

	return res, nil
}
