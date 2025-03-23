package activeflowhandler

import (
	"context"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *activeflowHandler) variableCreate(ctx context.Context, af *activeflow.Activeflow) (*variable.Variable, error) {
	log := logrus.WithFields(logrus.Fields{
		"activeflow_id": af.ID,
	})

	variables := map[string]string{}
	if af.ReferenceActiveflowID != uuid.Nil {
		log.Debugf("The reference_activeflow_id is not nil. Getting reference activeflow's variables. reference_activeflow_id: %s", af.ReferenceActiveflowID)

		tmp, err := h.variableHandler.Get(ctx, af.ReferenceActiveflowID)
		if err != nil || tmp == nil {
			// could not get the variable. but write the log only.
			log.Errorf("Could not get the variable. reference_activeflow_id: %s", af.ReferenceActiveflowID)
		} else {
			variables = tmp.Variables
		}
	}

	// overwrite the variables
	variables[variableActiveflowID] = af.ID.String()
	variables[variableActiveflowReferenceType] = string(af.ReferenceType)
	variables[variableActiveflowReferenceID] = af.ReferenceID.String()
	variables[variableActiveflowReferenceActiveflowID] = af.ReferenceActiveflowID.String()
	variables[variableActiveflowFlowID] = af.FlowID.String()

	res, err := h.variableHandler.Create(ctx, af.ID, variables)
	if err != nil {
		return nil, errors.Wrapf(err, "could not set the variable. activeflow_id: %s", af.ID)
	}

	return res, nil
}
