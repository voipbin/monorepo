package activeflowhandler

import (
	"context"
	"fmt"
	"maps"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *activeflowHandler) variableCreate(ctx context.Context, af *activeflow.Activeflow) (*variable.Variable, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "variableCreate",
		"activeflow_id": af.ID,
	})

	completeCount := 0
	variables := map[string]string{}
	if af.ReferenceActiveflowID != uuid.Nil {
		log.Debugf("The reference_activeflow_id is not nil. Getting reference activeflow's variables. reference_activeflow_id: %s", af.ReferenceActiveflowID)

		tmp, err := h.variableHandler.Get(ctx, af.ReferenceActiveflowID)
		if err != nil || tmp == nil {
			return nil, errors.Wrapf(err, "could not get the variable. reference_activeflow_id: %s", af.ReferenceActiveflowID)
		}

		// copy the variables
		maps.Copy(variables, tmp.Variables)

		// get the complete count
		val, ok := variables[variableActiveflowCompleteCount]
		if !ok {
			return nil, fmt.Errorf("could not find the complete count variable. activeflow_id: %s", af.ReferenceActiveflowID)
		}

		_, err = fmt.Sscanf(val, "%d", &completeCount)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse the complete count variable. value: %s, activeflow_id: %s", val, af.ReferenceActiveflowID)
		}

		completeCount++
		log.Debugf("The activeflow complete count is %d. activeflow_id: %s", completeCount, af.ReferenceActiveflowID)

		// check the max complete count
		if completeCount >= maxActiveflowCompleteCount {
			return nil, fmt.Errorf("the activeflow has reached the max complete count (%d). activeflow_id: %s", maxActiveflowCompleteCount, af.ID)
		}
	}

	// overwrite the variables
	variables[variableActiveflowID] = af.ID.String()
	variables[variableActiveflowReferenceType] = string(af.ReferenceType)
	variables[variableActiveflowReferenceID] = af.ReferenceID.String()
	variables[variableActiveflowReferenceActiveflowID] = af.ReferenceActiveflowID.String()
	variables[variableActiveflowFlowID] = af.FlowID.String()
	variables[variableActiveflowCompleteCount] = fmt.Sprintf("%d", completeCount)

	res, err := h.variableHandler.Create(ctx, af.ID, variables)
	if err != nil {
		return nil, errors.Wrapf(err, "could not set the variable. activeflow_id: %s", af.ID)
	}

	return res, nil
}
