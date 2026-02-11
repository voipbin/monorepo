package activeflowhandler

import (
	"context"
	"time"

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
		return nil, errors.Wrapf(err, "could not get activeflow info. activeflow_id: %s", id)
	}

	if af.Status == activeflow.StatusEnded {
		// already ended. nothing to do
		return af, nil
	}

	res, err := h.updateStatus(ctx, id, activeflow.StatusEnded)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update activeflow status. activeflow_id: %s, status: %s", id, activeflow.StatusEnded)
	}

	// metrics
	promActiveflowEndedTotal.WithLabelValues(string(af.ReferenceType)).Inc()
	promActiveflowRunning.WithLabelValues(string(af.ReferenceType)).Dec()
	if af.TMCreate != nil {
		promActiveflowDurationSeconds.WithLabelValues(string(af.ReferenceType)).Observe(time.Since(*af.TMCreate).Seconds())
	}

	// start on complete flow if configured
	if res.OnCompleteFlowID != uuid.Nil {
		log.Debugf("The activeflow has on_complete_flow configured. on_complete_flow_id: %s", res.OnCompleteFlowID)
		tmp, err := h.startOnCompleteFlow(ctx, res)
		if err != nil {
			log.Errorf("could not start on complete flow. activeflow_id: %s, err: %v", res.ID, err)
		} else if tmp != nil {
			log.WithField("new_activeflow", tmp).Debugf("Started on complete flow. new_activeflow_id: %s", tmp.ID)
		}
	}

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
