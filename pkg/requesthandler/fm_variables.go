package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	fmvariable "gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
	fmrequest "gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// FMV1VariableGet returns a variable.
func (r *requestHandler) FMV1VariableGet(ctx context.Context, variableID uuid.UUID) (*fmvariable.Variable, error) {

	uri := fmt.Sprintf("/v1/variables/%s", variableID)

	tmp, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodGet, resourceFMVariables, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	var res fmvariable.Variable
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FMV1VariableSetVariable sends a request to flow-manager
// to set the detail variable info.
// it returns updated variable info if it succeed.
func (r *requestHandler) FMV1VariableSetVariable(ctx context.Context, variableID uuid.UUID, key string, value string) (*fmvariable.Variable, error) {
	uri := fmt.Sprintf("/v1/variables/%s/variables", variableID)

	data := &fmrequest.V1DataVariablesIDVariablesPost{
		Key:   key,
		Value: value,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestFM(uri, rabbitmqhandler.RequestMethodPost, resourceFMVariables, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res fmvariable.Variable
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
