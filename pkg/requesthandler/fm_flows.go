package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/fmflow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// FMFlowCreate sends a request to flow-manager
// to creating a flow.
// it returns created flow if it succeed.
func (r *requestHandler) FMFlowCreate(userID uint64, id uuid.UUID, name, detail string, actions []action.Action, persist bool) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows")

	data := &request.FMV1DataFlowPost{
		ID:      id,
		UserID:  userID,
		Name:    name,
		Detail:  detail,
		Actions: actions,
		Persist: persist,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFlow(uri, rabbitmqhandler.RequestMethodPost, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CMFlowGet sends a request to flow-manager
// to getting a detail flow info.
// it returns detail flow info if it succeed.
func (r *requestHandler) FMFlowGet(flowID uuid.UUID) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", flowID)

	res, err := r.sendRequestFlow(uri, rabbitmqhandler.RequestMethodGet, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var f fmflow.Flow
	if err := json.Unmarshal([]byte(res.Data), &f); err != nil {
		return nil, err
	}

	return &f, nil
}

// FMFlowUpdate sends a request to flow-manager
// to update the detail flow info.
// it returns updated flow info if it succeed.
func (r *requestHandler) FMFlowUpdate(f *fmflow.Flow) (*fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows/%s", f.ID)

	data := &request.FMV1DataFlowIDPut{
		Name:    f.Name,
		Detail:  f.Detail,
		Actions: f.Actions,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestFlow(uri, rabbitmqhandler.RequestMethodPut, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resFlow fmflow.Flow
	if err := json.Unmarshal([]byte(res.Data), &resFlow); err != nil {
		return nil, err
	}

	return &resFlow, nil
}

// FMFlowGets sends a request to flow-manager
// to getting a list of flow info.
// it returns detail list of flow info if it succeed.
func (r *requestHandler) FMFlowGets(userID uint64, pageToken string, pageSize uint64) ([]fmflow.Flow, error) {
	uri := fmt.Sprintf("/v1/flows?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(pageToken), pageSize, userID)

	res, err := r.sendRequestFlow(uri, rabbitmqhandler.RequestMethodGet, resourceFlowFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var f []fmflow.Flow
	if err := json.Unmarshal([]byte(res.Data), &f); err != nil {
		return nil, err
	}

	return f, nil
}
