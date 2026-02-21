package requesthandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-flow-manager/models/action"

	uuid "github.com/gofrs/uuid"
)

// FlowV1ActionGet gets the action of the flow.
func (r *requestHandler) FlowV1ActionGet(ctx context.Context, flowID, actionID uuid.UUID) (*action.Action, error) {

	uri := fmt.Sprintf("/flows/%s/actions/%s", flowID, actionID)

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodGet, "flow/actions", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res action.Action
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
