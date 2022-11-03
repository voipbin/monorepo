package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	fmrequest "gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// FlowV1ActiveflowCreate creates a new activeflow.
func (r *requestHandler) FlowV1ActiveflowCreate(ctx context.Context, id, flowID uuid.UUID, referenceType fmactiveflow.ReferenceType, referenceID uuid.UUID) (*fmactiveflow.Activeflow, error) {

	uri := "/v1/activeflows"

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsPost{
		ID:            id,
		FlowID:        flowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceFlowActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get next action")
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1ActiveflowDelete delets activeflow.
func (r *requestHandler) FlowV1ActiveflowDelete(ctx context.Context, id uuid.UUID) (*fmactiveflow.Activeflow, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s", id)

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceFlowActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get next action")
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1ActiveflowGetNextAction gets the next action.
func (r *requestHandler) FlowV1ActiveflowGetNextAction(ctx context.Context, id, currentActionID uuid.UUID) (*fmaction.Action, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s/next", id)

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsIDNextGet{
		CurrentActionID: currentActionID,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceFlowActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get next action")
	}

	var res fmaction.Action
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1ActiveflowUpdateForwardActionID updates the forward action id.
func (r *requestHandler) FlowV1ActiveflowUpdateForwardActionID(ctx context.Context, id, forwardActionID uuid.UUID, forwardNow bool) error {

	uri := fmt.Sprintf("/v1/activeflows/%s/forward_action_id", id)

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsIDForwardActionIDPut{
		ForwardActionID: forwardActionID,
		ForwardNow:      forwardNow,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceFlowActiveFlows, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("could not get next action")
	}

	return nil
}

// FlowV1ActiveflowExecute executes the activeflow
func (r *requestHandler) FlowV1ActiveflowExecute(ctx context.Context, id uuid.UUID) error {

	uri := fmt.Sprintf("/v1/activeflows/%s/execute", id)

	res, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceFlowActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}
