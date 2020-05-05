package requesthandler

import (
	"encoding/json"
	"fmt"

	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

func (r *requestHandler) FlowActionGet(flowID, actionID uuid.UUID) (*action.Action, error) {

	uri := fmt.Sprintf("/flows/%s/actions/%s", flowID, actionID)

	res, err := r.sendRequestFlow(uri, rabbitmq.RequestMethodGet, requestTimeoutDefault, ContentTypeJSON, "")
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
