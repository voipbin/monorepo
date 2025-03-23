package activeflowhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// variableSubstitueAddress substitue the address with variables
func (h *activeflowHandler) variableSubstitueAddress(ctx context.Context, address *commonaddress.Address, v *variable.Variable) {
	address.Name = h.variableHandler.SubstituteString(ctx, address.Name, v)
	address.Detail = h.variableHandler.SubstituteString(ctx, address.Detail, v)
	address.Target = h.variableHandler.SubstituteString(ctx, address.Target, v)
	address.TargetName = h.variableHandler.SubstituteString(ctx, address.TargetName, v)
}

func (h *activeflowHandler) variableCreate(ctx context.Context, af *activeflow.Activeflow, referenceActiveflowID uuid.UUID) (*variable.Variable, error) {
	log := logrus.WithFields(logrus.Fields{
		"activeflow_id":           af.ID,
		"reference_activeflow_id": referenceActiveflowID,
	})

	variables := map[string]string{}
	if referenceActiveflowID != uuid.Nil {
		log.Debugf("The reference_activeflow_id is not nil. Getting reference activeflow's variables. reference_activeflow_id: %s", referenceActiveflowID)

		tmp, err := h.variableHandler.Get(ctx, referenceActiveflowID)
		if err != nil || tmp == nil {
			// could not get the variable. but write the log only.
			log.Errorf("Could not get the variable. reference_activeflow_id: %s", referenceActiveflowID)
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
