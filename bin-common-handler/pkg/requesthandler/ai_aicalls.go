package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	ammessage "monorepo/bin-ai-manager/models/message"
	cbrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (r *requestHandler) AIV1AIcallStart(ctx context.Context, activeflowID uuid.UUID, aiID uuid.UUID, referenceType amaicall.ReferenceType, referenceID uuid.UUID, gender amaicall.Gender, language string) (*amaicall.AIcall, error) {
	uri := "/v1/aicalls"

	data := &cbrequest.V1DataAIcallsPost{
		ActiveflowID: activeflowID,

		AIID: aiID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Gender:   gender,
		Language: language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallGets sends a request to ai-manager
// to getting a list of aicall info of the given customer id.
// it returns detail list of aicall info if it succeed.
func (r *requestHandler) AIV1AIcallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[amaicall.Field]any) ([]amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AIcallGet returns the aicall.
func (r *requestHandler) AIV1AIcallGet(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {

	uri := fmt.Sprintf("/v1/aicalls/%s", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls/<aicall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallDelete sends a request to ai-manager
// to deleting a aicall.
// it returns deleted aicall if it succeed.
func (r *requestHandler) AIV1AIcallDelete(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/aicalls/<aicall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallTerminate sends a request to ai-manager
// to terminate an aicall.
// it returns aicall if it succeed.
func (r *requestHandler) AIV1AIcallTerminate(ctx context.Context, aicallID uuid.UUID) (*amaicall.AIcall, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s/terminate", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls/<aicall-id>/terminate", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res amaicall.AIcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1AIcallTerminateWithDelay sends a request to ai-manager
// to terminate an aicall after delayed time.
// it returns null if it succeed.
func (r *requestHandler) AIV1AIcallTerminateWithDelay(ctx context.Context, aicallID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/aicalls/%s/terminate", aicallID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls/<aicall-id>/terminate", requestTimeoutDefault, delay, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AIV1AIcallToolExecute sends a request to ai-manager
// to execute the tool on the given aicall.
// it returns response message if it succeed.
func (r *requestHandler) AIV1AIcallToolExecute(
	ctx context.Context,
	aicallID uuid.UUID,
	toolID string,
	toolType ammessage.ToolType,
	function *ammessage.FunctionCall,
) (map[string]any, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s/tool_execute", aicallID)

	data := &cbrequest.V1DataAIcallsIDToolExecutePost{
		ID: toolID,

		Type:     toolType,
		Function: *function,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/aicalls/<aicall-id>/tool_execute", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res map[string]any
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
