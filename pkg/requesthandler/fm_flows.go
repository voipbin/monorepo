package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	fmrequest "gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"
)

func (r *requestHandler) FMFlowsPost(userID uint64, name string, detail string, webhookURI string, actions []action.Action, persist bool) (*fmflow.Flow, error) {

	uri := "/v1/flows"

	reqData := &fmrequest.V1DataFlowPost{
		UserID:     userID,
		Name:       name,
		Detail:     detail,
		Actions:    actions,
		WebhookURI: webhookURI,
		Persist:    persist,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestFlow(uri, rabbitmqhandler.RequestMethodPost, resourceFlowsActions, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	var resFlow fmflow.Flow
	if err := json.Unmarshal([]byte(res.Data), &resFlow); err != nil {
		return nil, err
	}

	return &resFlow, nil
}
