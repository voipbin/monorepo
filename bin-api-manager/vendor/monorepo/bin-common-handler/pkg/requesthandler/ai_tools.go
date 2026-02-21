package requesthandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	amtool "monorepo/bin-ai-manager/models/tool"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

// V1ToolsGet is the response structure for GET /v1/tools
type V1ToolsGetResponse struct {
	Tools []amtool.Tool `json:"tools"`
}

// AIV1ToolList retrieves all tools from ai-manager
func (r *requestHandler) AIV1ToolList(ctx context.Context) ([]amtool.Tool, error) {
	log := logrus.WithField("func", "AIV1ToolList")

	// Send request
	res, err := r.SendRequest(
		ctx,
		commonoutline.QueueNameAIRequest,
		"/v1/tools",
		sock.RequestMethodGet,
		requestTimeoutDefault,
		0,
		"",
		nil,
	)
	if err != nil {
		log.Errorf("Could not get tools. err: %v", err)
		return nil, err
	}

	var response V1ToolsGetResponse
	if err := json.Unmarshal(res.Data, &response); err != nil {
		log.Errorf("Could not unmarshal response. err: %v", err)
		return nil, err
	}

	return response.Tools, nil
}
