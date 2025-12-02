package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
	fmrequest "monorepo/bin-flow-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// FlowV1FlowCreate creates a new flow.
func (r *requestHandler) FlowV1FlowCreate(
	ctx context.Context,
	customerID uuid.UUID,
	flowType fmflow.Type,
	name string,
	detail string,
	actions []fmaction.Action,
	onCompleteFlowID uuid.UUID,
	persist bool,
) (*fmflow.Flow, error) {

	uri := "/v1/flows"

	reqData := &fmrequest.V1DataFlowsPost{
		CustomerID:       customerID,
		Type:             flowType,
		Name:             name,
		Detail:           detail,
		Actions:          actions,
		OnCompleteFlowID: onCompleteFlowID,
		Persist:          persist,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/actions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res fmflow.Flow
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// FlowV1FlowGet sends a request to flow-manager
// to getting a detail flow info.
// it returns detail flow info if it succeed.
func (r *requestHandler) FlowV1FlowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", flowID)

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodGet, "flow/flows", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res fmflow.Flow
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// FlowV1FlowDelete sends a request to flow-manager
// to deleting the flow.
func (r *requestHandler) FlowV1FlowDelete(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", flowID)

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodDelete, "flow/flows", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res fmflow.Flow
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// FlowV1FlowUpdate sends a request to flow-manager
// to update the detail flow info.
// it returns updated flow info if it succeed.
func (r *requestHandler) FlowV1FlowUpdate(
	ctx context.Context,
	f *fmflow.Flow,
) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", f.ID)

	data := &fmrequest.V1DataFlowsIDPut{
		Name:             f.Name,
		Detail:           f.Detail,
		Actions:          f.Actions,
		OnCompleteFlowID: f.OnCompleteFlowID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPut, "flow/flows", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res fmflow.Flow
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPut, "flow/flows", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res fmflow.Flow
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// FlowV1FlowGets sends a request to flow-manager
// to getting a list of flows.
// it returns detail list of flows if it succeed.
func (r *requestHandler) FlowV1FlowGets(ctx context.Context, pageToken string, pageSize uint64, filters map[fmflow.Field]any) ([]fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodGet, "flow/flows", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []fmflow.Flow
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
