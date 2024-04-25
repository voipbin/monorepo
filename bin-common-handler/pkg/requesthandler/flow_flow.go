package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
	fmrequest "monorepo/bin-flow-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// FlowV1FlowCreate creates a new flow.
func (r *requestHandler) FlowV1FlowCreate(ctx context.Context, customerID uuid.UUID, flowType fmflow.Type, name string, detail string, actions []fmaction.Action, persist bool) (*fmflow.Flow, error) {

	uri := "/v1/flows"

	reqData := &fmrequest.V1DataFlowPost{
		CustomerID: customerID,
		Type:       flowType,
		Name:       name,
		Detail:     detail,
		Actions:    actions,
		Persist:    persist,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceFlowActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not create flow. status: %d", tmp.StatusCode)
	}

	var res fmflow.Flow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1FlowGet sends a request to flow-manager
// to getting a detail flow info.
// it returns detail flow info if it succeed.
func (r *requestHandler) FlowV1FlowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", flowID)

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res fmflow.Flow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1FlowDelete sends a request to flow-manager
// to deleting the flow.
func (r *requestHandler) FlowV1FlowDelete(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", flowID)

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res fmflow.Flow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1FlowUpdate sends a request to flow-manager
// to update the detail flow info.
// it returns updated flow info if it succeed.
func (r *requestHandler) FlowV1FlowUpdate(ctx context.Context, f *fmflow.Flow) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", f.ID)

	data := &fmrequest.V1DataFlowIDPut{
		Name:    f.Name,
		Detail:  f.Detail,
		Actions: f.Actions,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res fmflow.Flow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1FlowUpdateActions sends a request to flow-manager
// to update the actions.
// it returns updated flow info if it succeed.
func (r *requestHandler) FlowV1FlowUpdateActions(ctx context.Context, flowID uuid.UUID, actions []fmaction.Action) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s/actions", flowID)

	data := &fmrequest.V1DataFlowIDActionsPut{
		Actions: actions,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res fmflow.Flow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1FlowGets sends a request to flow-manager
// to getting a list of flows.
// it returns detail list of flows if it succeed.
func (r *requestHandler) FlowV1FlowGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []fmflow.Flow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}
