package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-flow-manager/models/action"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// FlowV1ActionGet gets the action of the flow.
func (r *requestHandler) FlowV1ActionGet(ctx context.Context, flowID, actionID uuid.UUID) (*action.Action, error) {

	uri := fmt.Sprintf("/flows/%s/actions/%s", flowID, actionID)

	res, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceFlowActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get action. status: %d", res.StatusCode)
	}

	var action action.Action
	if err := json.Unmarshal([]byte(res.Data), &action); err != nil {
		return nil, err
	}

	return &action, nil
}
