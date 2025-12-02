package activeflowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-flow-manager/models/action"
)

// actionGetsFromFlow gets the actions from the flow.
func (h *activeflowHandler) actionGetsFromFlow(ctx context.Context, flowID uuid.UUID, customerID uuid.UUID) ([]action.Action, error) {

	f, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get flow. flow_id: %s", flowID)
	}

	if f.CustomerID != customerID {
		return nil, fmt.Errorf("the customer has no permission. customer_id: %s", customerID)
	}

	return f.Actions, nil
}
