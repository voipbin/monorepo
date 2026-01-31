package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-ai-manager/pkg/listenhandler/models/response"
	"monorepo/bin-common-handler/models/sock"
)

// processV1ToolsGet handles GET /v1/tools
func (h *listenHandler) processV1ToolsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	tools := h.toolHandler.GetAll()

	res := &response.V1ToolsGet{
		Tools: tools,
	}

	data, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
