package activeflowhandler

import (
	"context"
	"fmt"
	"maps"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *activeflowHandler) variableCreate(ctx context.Context, af *activeflow.Activeflow) (*variable.Variable, error) {
	completeCount := 0
	variables := map[string]string{}
	if af.ReferenceActiveflowID != uuid.Nil {
		if errSet := h.variableSetFromReferenceActiveflow(ctx, variables, af.ReferenceActiveflowID); errSet != nil {
			return nil, errors.Wrapf(errSet, "could not set variable from reference activeflow. reference_activeflow_id: %s", af.ReferenceActiveflowID)
		}

		tmpCompleteCount, err := h.variableParseCompleteCount(variables)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse complete count from variable. reference_activeflow_id: %s", af.ReferenceActiveflowID)
		}
		tmpCompleteCount++

		if tmpCompleteCount >= maxActiveflowCompleteCount {
			return nil, fmt.Errorf("the activeflow has reached the max complete count (%d). activeflow_id: %s", maxActiveflowCompleteCount, af.ID)
		}

		completeCount = tmpCompleteCount
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

func (h *activeflowHandler) variableSetFromReferenceActiveflow(ctx context.Context, variables map[string]string, referenceActiveflowID uuid.UUID) error {
	tmp, err := h.variableHandler.Get(ctx, referenceActiveflowID)
	if err != nil {
		return errors.Wrapf(err, "could not get the variable. reference_activeflow_id: %s", referenceActiveflowID)
	}

	// copy the variables
	maps.Copy(variables, tmp.Variables)

	return nil
}

func (h *activeflowHandler) variableParseCompleteCount(variables map[string]string) (int, error) {
	val, ok := variables[variableActiveflowCompleteCount]
	if !ok {
		return 0, fmt.Errorf("could not find the complete count variable")
	}

	res := 0
	_, err := fmt.Sscanf(val, "%d", &res)
	if err != nil {
		return 0, errors.Wrapf(err, "could not parse the complete count variable. value: %s ", val)
	}

	return res, nil
}
