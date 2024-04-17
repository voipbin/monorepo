package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmrequest "monorepo/bin-flow-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// FlowV1ActiveflowCreate creates a new activeflow.
func (r *requestHandler) FlowV1ActiveflowCreate(ctx context.Context, activeflowID, flowID uuid.UUID, referenceType fmactiveflow.ReferenceType, referenceID uuid.UUID) (*fmactiveflow.Activeflow, error) {

	uri := "/v1/activeflows"

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsPost{
		ID:            activeflowID,
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

// FlowV1ActiveflowGet sends a request to flow-manager
// to getting a detail activeflow info.
// it returns detail activeflow info if it succeed.
func (r *requestHandler) FlowV1ActiveflowGet(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error) {
	uri := fmt.Sprintf("/v1/activeflows/%s", activeflowID)

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceFlowActiveFlows, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1ActiveflowGets sends a request to flow-manager
// to getting a list of activeflow info.
// it returns detail list of activeflow info if it succeed.
func (r *requestHandler) FlowV1ActiveflowGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]fmactiveflow.Activeflow, error) {
	uri := fmt.Sprintf("/v1/activeflows?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = parseFilters(uri, filters)

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallCalls, 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// FlowV1ActiveflowDelete delets activeflow.
func (r *requestHandler) FlowV1ActiveflowDelete(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s", activeflowID)

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
func (r *requestHandler) FlowV1ActiveflowGetNextAction(ctx context.Context, activeflowID, currentActionID uuid.UUID) (*fmaction.Action, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s/next", activeflowID)

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
func (r *requestHandler) FlowV1ActiveflowUpdateForwardActionID(ctx context.Context, activeflowID, forwardActionID uuid.UUID, forwardNow bool) error {

	uri := fmt.Sprintf("/v1/activeflows/%s/forward_action_id", activeflowID)

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
func (r *requestHandler) FlowV1ActiveflowExecute(ctx context.Context, activeflowID uuid.UUID) error {

	uri := fmt.Sprintf("/v1/activeflows/%s/execute", activeflowID)

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

// FlowV1ActiveflowStop stops activeflow.
func (r *requestHandler) FlowV1ActiveflowStop(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s/stop", activeflowID)

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceFlowActions, requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not stop the activeflow")
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1ActiveflowPushActions pushes actions to next to the current action of the given activeflow.
func (r *requestHandler) FlowV1ActiveflowPushActions(ctx context.Context, activeflowID uuid.UUID, actions []fmaction.Action) (*fmactiveflow.Activeflow, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s/push_actions", activeflowID)

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsIDPushActionPost{
		Actions: actions,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPost, "flow/activeflows/<activeflow-id>/push_actions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not stop the activeflow")
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
