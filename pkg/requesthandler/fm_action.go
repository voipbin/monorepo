package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// FMV1ActionGet gets the action of the flow.
func (r *requestHandler) FMV1ActionGet(ctx context.Context, flowID, actionID uuid.UUID) (*action.Action, error) {

	uri := fmt.Sprintf("/flows/%s/actions/%s", flowID, actionID)

	res, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodGet, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	var action action.Action
	if err := json.Unmarshal([]byte(res.Data), &action); err != nil {
		return nil, err
	}

	return &action, nil
}
