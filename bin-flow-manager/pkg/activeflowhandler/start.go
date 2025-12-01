package activeflowhandler

import (
	"context"
	"monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *activeflowHandler) startOnCompleteFlow(ctx context.Context, af *activeflow.Activeflow) (*activeflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startOnCompleteFlow",
	})

	// create a new activeflow
	res, err := h.Create(
		ctx,
		uuid.Nil,
		af.CustomerID,
		af.ReferenceType,
		af.ReferenceID,
		af.ID,
		af.OnCompleteFlowID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create on complete activeflow. flow_id: %s", af.OnCompleteFlowID)
	}
	log.WithField("activeflow", res).Debugf("Created a new activeflow. activeflow_id: %s", res.ID)

	// execute
	go func() {
		if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, res.ID); errExecute != nil {
			log.Errorf("could not execute the on complete activeflow. activeflow_id: %s, err: %v", res.ID, errExecute)
			return
		}
		log.Debugf("Executed the on complete activeflow. activeflow_id: %s", res.ID)
	}()

	return res, nil
}
