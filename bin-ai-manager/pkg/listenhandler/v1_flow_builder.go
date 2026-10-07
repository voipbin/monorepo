package listenhandler

import (
	"context"
	"encoding/json"
	"net/http"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

// processFlowBuilderChatPost handles /v1/flow_builder/chat. It is reached
// only through processBuilder, so it inherits that function's contract: m and
// m.Data are never logged, every failure becomes (response, nil) via
// errorResponse, and a panic becomes a bare 500.
func (h *listenHandler) processFlowBuilderChatPost(ctx context.Context, m *sock.Request) *sock.Response {
	var req request.V1DataFlowBuilderChatPost
	if errUnmarshal := json.Unmarshal(m.Data, &req); errUnmarshal != nil {
		// The error text is not used: json errors can quote the offending bytes.
		return errorResponse(cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "the request body could not be read"))
	}
	if req.CustomerID.IsNil() {
		return errorResponse(cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "the customer is missing"))
	}

	res, errChat := h.flowBuilderHandler.Chat(ctx, req.CustomerID, &flowbuilder.ChatRequest{
		Messages:             req.Messages,
		CurrentDraft:         req.CurrentDraft,
		SupportedActionTypes: req.SupportedActionTypes,
	})
	if errChat != nil {
		return errorResponse(errChat)
	}

	data, errMarshal := json.Marshal(res)
	if errMarshal != nil {
		return errorResponse(cerrors.Internal(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the response could not be written"))
	}

	return &sock.Response{StatusCode: http.StatusOK, DataType: "application/json", Data: data}
}
