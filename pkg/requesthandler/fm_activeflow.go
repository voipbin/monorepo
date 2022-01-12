package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	fmrequest "gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"
)

// FMV1ActvieFlowCreate creates a new active-flow.
func (r *requestHandler) FMV1ActvieFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error) {

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

	res, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodPost, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// FMV1ActvieFlowGetNextAction gets the next action.
func (r *requestHandler) FMV1ActvieFlowGetNextAction(ctx context.Context, callID, currentActionID uuid.UUID) (*action.Action, error) {

	uri := fmt.Sprintf("/v1/active-flows/%s/next", callID)

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsIDNextGet{
		CurrentActionID: currentActionID,
	})
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodGet, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// FMV1ActvieFlowUpdateForwardActionID updates the forward action id.
func (r *requestHandler) FMV1ActvieFlowUpdateForwardActionID(ctx context.Context, callID, forwardActionID uuid.UUID, forwardNow bool) error {

	uri := fmt.Sprintf("/v1/active-flows/%s/forward_action_id", callID)
	m, err := json.Marshal(fmrequest.V1DataActiveFlowsIDForwardActionIDPut{
		ForwardActionID: forwardActionID,
		ForwardNow:      forwardNow,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodPut, resourceFMActiveFlows, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("could not get next action")
	}

	return nil
}
