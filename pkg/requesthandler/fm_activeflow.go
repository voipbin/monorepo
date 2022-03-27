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

// FMV1ActiveflowCreate creates a new activeflow.
func (r *requestHandler) FMV1ActiveflowCreate(ctx context.Context, flowID uuid.UUID, referenceType fmactiveflow.ReferenceType, referenceID uuid.UUID) (*fmactiveflow.Activeflow, error) {

	uri := "/v1/activeflows"

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsPost{
		FlowID:        flowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
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

	var af fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(res.Data), &af); err != nil {
		return nil, err
	}

	return &af, nil
}

// FMV1ActiveflowDelete delets activeflow.
func (r *requestHandler) FMV1ActiveflowDelete(ctx context.Context, id uuid.UUID) (*fmactiveflow.Activeflow, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s", id)

	res, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodDelete, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get next action")
	}

	var af fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(res.Data), &af); err != nil {
		return nil, err
	}

	return &af, nil
}

// FMV1ActiveflowGetNextAction gets the next action.
func (r *requestHandler) FMV1ActiveflowGetNextAction(ctx context.Context, id, currentActionID uuid.UUID) (*fmaction.Action, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s/next", id)

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

	var action fmaction.Action
	if err := json.Unmarshal([]byte(res.Data), &action); err != nil {
		return nil, err
	}

	return &action, nil
}

// FMV1ActiveflowUpdateForwardActionID updates the forward action id.
func (r *requestHandler) FMV1ActiveflowUpdateForwardActionID(ctx context.Context, id, forwardActionID uuid.UUID, forwardNow bool) error {

	uri := fmt.Sprintf("/v1/activeflows/%s/forward_action_id", id)

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

// FMV1ActiveflowExecute executes the activeflow
func (r *requestHandler) FMV1ActiveflowExecute(ctx context.Context, id uuid.UUID) error {

	uri := fmt.Sprintf("/v1/activeflows/%s/execute", id)

	res, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodPost, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
