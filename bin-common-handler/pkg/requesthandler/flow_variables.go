package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	flowVariable "monorepo/bin-flow-manager/models/variable"
	fmrequest "monorepo/bin-flow-manager/pkg/listenhandler/models/request"
	fmresponse "monorepo/bin-flow-manager/pkg/listenhandler/models/response"

	"github.com/gofrs/uuid"
)

// FlowV1VariableGet returns a variable.
func (r *requestHandler) FlowV1VariableGet(ctx context.Context, variableID uuid.UUID) (*flowVariable.Variable, error) {

	uri := fmt.Sprintf("/v1/variables/%s", variableID)

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodGet, "flow/variables", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res flowVariable.Variable
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/variables", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// FlowV1VariableDeleteVariable sends a request to flow-manager
// to delete the variable info.
func (r *requestHandler) FlowV1VariableDeleteVariable(ctx context.Context, variableID uuid.UUID, key string) error {
	uri := fmt.Sprintf("/v1/variables/%s/variables/%s", variableID, url.QueryEscape(key))

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodDelete, "flow/variables", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// FlowV1VariableSubstitute sends a request to flow-manager
// to substitute the data with variable info.
// it returns error if it failed.
func (r *requestHandler) FlowV1VariableSubstitute(ctx context.Context, variableID uuid.UUID, dataString string) (string, error) {
	uri := fmt.Sprintf("/v1/variables/%s/substitute", variableID)

	data := &fmrequest.V1DataVariablesIDSubstitutePost{
		Data: dataString,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	tmp, err := r.sendRequestFlow(ctx, uri, sock.RequestMethodPost, "flow/variables/<variable-id>/substitute", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return "", err
	}

	var res fmresponse.V1ResponseVariablesIDSubstitutePost
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return "", errParse
	}

	return res.Data, nil
}
