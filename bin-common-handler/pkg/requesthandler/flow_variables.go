package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	flowVariable "gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
	fmrequest "gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// FlowV1VariableGet returns a variable.
func (r *requestHandler) FlowV1VariableGet(ctx context.Context, variableID uuid.UUID) (*flowVariable.Variable, error) {

	uri := fmt.Sprintf("/v1/variables/%s", variableID)

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceFlowVariables, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get variable. status: %d", tmp.StatusCode)
	}

	var res flowVariable.Variable
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// FlowV1VariableSetVariable sends a request to flow-manager
// to set the detail variable info.
// it returns error if it failed.
func (r *requestHandler) FlowV1VariableSetVariable(ctx context.Context, variableID uuid.UUID, variables map[string]string) error {
	uri := fmt.Sprintf("/v1/variables/%s/variables", variableID)

	data := &fmrequest.V1DataVariablesIDVariablesPost{
		Variables: variables,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceFlowVariables, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// FlowV1VariableDeleteVariable sends a request to flow-manager
// to delete the variable info.
func (r *requestHandler) FlowV1VariableDeleteVariable(ctx context.Context, variableID uuid.UUID, key string) error {
	uri := fmt.Sprintf("/v1/variables/%s/variables/%s", variableID, url.QueryEscape(key))

	tmp, err := r.sendRequestFlow(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceFlowVariables, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
