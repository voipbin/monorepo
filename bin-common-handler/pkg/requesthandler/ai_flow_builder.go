package requesthandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/models/flowbuilder"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
)

// AIV1FlowBuilderChat sends one Flow Builder turn to ai-manager and returns
// the model's reply and the confirmed draft.
//
// It waits as long as the Assistant Builder does (builderChatTimeout): the
// LLM deadline, the RPC wait and the client wait keep the same order. The
// resource label "ai/flow_builder/chat" only names the duration series; the
// circuit breaker is per queue and is shared with every other ai-manager RPC
// (design doc 5.1, accepted coupling).
//
// customerID is the authenticated customer, put into the request so ai-manager
// charges its daily counter to the right account; the browser never supplies it.
// Errors from ai-manager come back as a *VoipbinError with the Builder reason
// intact. Nothing the customer wrote is placed in an error here.
func (r *requestHandler) AIV1FlowBuilderChat(ctx context.Context, customerID uuid.UUID, req *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error) {
	m, err := json.Marshal(&amrequest.V1DataFlowBuilderChatPost{
		CustomerID:           customerID,
		Messages:             req.Messages,
		CurrentDraft:         req.CurrentDraft,
		SupportedActionTypes: req.SupportedActionTypes,
	})
	if err != nil {
		// json errors can quote the value; this one is not wrapped with it.
		return nil, errors.New("could not marshal the flow builder chat request")
	}

	tmp, err := r.sendRequestAI(ctx, flowbuilder.URIChat, sock.RequestMethodPost, "ai/flow_builder/chat", builderChatTimeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res flowbuilder.ChatResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
