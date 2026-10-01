package requesthandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-ai-manager/models/builder"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
)

// Timeouts of the Assistant Builder RPCs, in milliseconds (the unit sendRequest
// takes).
//
// Chat waits 55 seconds. ai-manager's own LLM deadline is 40 seconds, so a slow
// but normal answer is not cut off here first. ai-manager's configuration
// validation keeps that deadline at 50 seconds or less, which is these 55 minus
// 5 seconds of headroom for queueing and parsing. Status is a plain read and
// fails fast.
const (
	builderChatTimeout   = 55000
	builderStatusTimeout = 3000
)

// AIV1BuilderChat sends one Assistant Builder turn to ai-manager and returns
// the model's reply.
//
// customerID is the authenticated customer, put into the request so ai-manager
// charges its daily counter to the right account; the browser never supplies it.
//
// Errors from ai-manager come back as a *VoipbinError with the Builder reason
// intact. Nothing the customer wrote is placed in an error here.
func (r *requestHandler) AIV1BuilderChat(ctx context.Context, customerID uuid.UUID, req *builder.ChatRequest) (*builder.ChatResponse, error) {
	m, err := json.Marshal(&amrequest.V1DataBuilderChatPost{
		CustomerID:   customerID,
		Messages:     req.Messages,
		CurrentDraft: req.CurrentDraft,
	})
	if err != nil {
		// json errors can quote the value; this one is not wrapped with it.
		return nil, errors.New("could not marshal the builder chat request")
	}

	tmp, err := r.sendRequestAI(ctx, builder.URIChat, sock.RequestMethodPost, "ai/ai_builder/chat", builderChatTimeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res builder.ChatResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1BuilderStatus asks ai-manager whether the Assistant Builder is usable and
// what its input limits are.
func (r *requestHandler) AIV1BuilderStatus(ctx context.Context) (*builder.StatusResponse, error) {
	tmp, err := r.sendRequestAI(ctx, builder.URIStatus, sock.RequestMethodGet, "ai/ai_builder/status", builderStatusTimeout, 0, "", nil)
	if err != nil {
		return nil, err
	}

	var res builder.StatusResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
