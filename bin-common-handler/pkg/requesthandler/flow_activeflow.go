package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmrequest "monorepo/bin-flow-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// FlowV1ActiveflowCreate creates a new activeflow.
func (r *requestHandler) FlowV1ActiveflowCreate(
	ctx context.Context,
	activeflowID uuid.UUID,
	customerID uuid.UUID,
	flowID uuid.UUID,
	referenceType fmactiveflow.ReferenceType,
	referenceID uuid.UUID,
	referenceActiveflowID uuid.UUID,
) (*fmactiveflow.Activeflow, error) {

	uri := "/v1/activeflows"

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsPost{
		ID:                    activeflowID,
		CustomerID:            customerID,
		FlowID:                flowID,
		ReferenceType:         referenceType,
		ReferenceID:           referenceID,
		ReferenceActiveflowID: referenceActiveflowID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the request")
	}

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/actions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the request")
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not create the activeflow. status_code: %d", tmp.StatusCode)
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal the response")
	}

	return &res, nil
}

// FlowV1ActiveflowGet sends a request to flow-manager
// to getting a detail activeflow info.
// it returns detail activeflow info if it succeed.
func (r *requestHandler) FlowV1ActiveflowGet(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error) {
	uri := fmt.Sprintf("/v1/activeflows/%s", activeflowID)

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodGet, "flow/activeflows", requestTimeoutDefault, 0, ContentTypeNone, nil)
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
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodGet, "call/calls", 30000, 0, ContentTypeNone, nil)
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

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodDelete, "flow/actions", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodGet, "flow/actions", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

	res, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPut, "flow/activeflows", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

	res, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/actions", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/actions", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the request")
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not stop the activeflow")
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal the response")
	}

	return &res, nil
}

// FlowV1ActiveflowAddActions adds actions to next to the current action and current stack of the given activeflow.
func (r *requestHandler) FlowV1ActiveflowAddActions(ctx context.Context, activeflowID uuid.UUID, actions []fmaction.Action) (*fmactiveflow.Activeflow, error) {

	uri := fmt.Sprintf("/v1/activeflows/%s/add_actions", activeflowID)

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsIDAddActionPost{
		Actions: actions,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the request")
	}

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/activeflows/<activeflow-id>/add_actions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, errors.Wrapf(err, "could not add the activeflow")
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not add the activeflow")
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal the response")
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
		return nil, errors.Wrapf(err, "could not marshal the request")
	}

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/activeflows/<activeflow-id>/push_actions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the request")
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not push the actions")
	}

	var res fmactiveflow.Activeflow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal the response")
	}

	return &res, nil
}

// FlowV1ActiveflowPushActions pushes actions to next to the current action of the given activeflow.
func (r *requestHandler) FlowV1ActiveflowServiceStop(ctx context.Context, activeflowID uuid.UUID, serviceID uuid.UUID) error {

	uri := fmt.Sprintf("/v1/activeflows/%s/service_stop", activeflowID)

	m, err := json.Marshal(fmrequest.V1DataActiveFlowsIDServiceStopPost{
		ServiceID: serviceID,
	})
	if err != nil {
		return errors.Wrapf(err, "could not marshal the request")
	}

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/activeflows/<activeflow-id>/service_stop", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return errors.Wrapf(err, "could not send the request")
	}

	if tmp.StatusCode >= 299 {
		return fmt.Errorf("could not stop the service")
	}

	return nil
}
