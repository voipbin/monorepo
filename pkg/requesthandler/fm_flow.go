package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	fmrequest "gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// FMV1FlowCreate creates a new flow.
func (r *requestHandler) FMV1FlowCreate(ctx context.Context, customerID uuid.UUID, flowType fmflow.Type, name string, detail string, actions []fmaction.Action, persist bool) (*fmflow.Flow, error) {

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

	tmp, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodPost, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	var res fmflow.Flow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FMV1FlowGet sends a request to flow-manager
// to getting a detail flow info.
// it returns detail flow info if it succeed.
func (r *requestHandler) FMV1FlowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", flowID)

	tmp, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodGet, resourceFMFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// FMV1FlowDelete sends a request to flow-manager
// to deleting the flow.
func (r *requestHandler) FMV1FlowDelete(ctx context.Context, flowID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/flows/%s", flowID)

	res, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodDelete, resourceFMFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}

// FMV1FlowUpdate sends a request to flow-manager
// to update the detail flow info.
// it returns updated flow info if it succeed.
func (r *requestHandler) FMV1FlowUpdate(ctx context.Context, f *fmflow.Flow) (*fmflow.Flow, error) {
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

	tmp, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodPut, resourceFMFlows, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// FMV1FlowUpdateActions sends a request to flow-manager
// to update the actions.
// it returns updated flow info if it succeed.
func (r *requestHandler) FMV1FlowUpdateActions(ctx context.Context, flowID uuid.UUID, actions []fmaction.Action) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s/actions", flowID)

	data := &fmrequest.V1DataFlowIDActionsPut{
		Actions: actions,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodPut, resourceFMFlows, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// FMV1FlowGets sends a request to flow-manager
// to getting a list of flows.
// it returns detail list of flows if it succeed.
func (r *requestHandler) FMV1FlowGets(ctx context.Context, customerID uuid.UUID, flowType fmflow.Type, pageToken string, pageSize uint64) ([]fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows?page_token=%s&page_size=%d&customer_id=%s&type=%s", url.QueryEscape(pageToken), pageSize, customerID, flowType)

	tmp, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodGet, resourceFMFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
