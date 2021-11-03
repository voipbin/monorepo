package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

func (r *requestHandler) FlowActvieFlowPost(callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error) {

	uri := "/v1/active-flows"

	type Data struct {
		CallID uuid.UUID `json:"call_id"`
		FlowID uuid.UUID `json:"flow_id"`
	}

	m, err := json.Marshal(Data{
		callID,
		flowID,
	})
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestFlow(uri, rabbitmqhandler.RequestMethodPost, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get next action")
	}

	var af activeflow.ActiveFlow
	if err := json.Unmarshal([]byte(res.Data), &af); err != nil {
		return nil, err
	}

	return &af, nil
}

func (r *requestHandler) FlowActvieFlowNextGet(callID, actionID uuid.UUID) (*action.Action, error) {

	uri := fmt.Sprintf("/v1/active-flows/%s/next", callID)

	type Data struct {
		CurrentActionID uuid.UUID `json:"current_action_id"`
	}

	m, err := json.Marshal(Data{
		actionID,
	})
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestFlow(uri, rabbitmqhandler.RequestMethodGet, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get next action")
	}

	var action action.Action
	if err := json.Unmarshal([]byte(res.Data), &action); err != nil {
		return nil, err
	}

	return &action, nil
}
