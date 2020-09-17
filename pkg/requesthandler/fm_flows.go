package requesthandler

import (
	"encoding/json"
	"fmt"

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
