package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	fmrequest "gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"
)

// FMFlowCreate sends a request to flow-manager
// to creating a flow.
// it returns created flow if it succeed.
func (r *requestHandler) FMFlowCreate(userID uint64, name, detail string, actions []action.Action, persist bool) (*flow.Flow, error) {
	uri := "/v1/flows"

	data := &fmrequest.V1DataFlowPost{
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

	var res flow.Flow
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
