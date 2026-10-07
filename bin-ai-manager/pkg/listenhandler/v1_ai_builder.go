package listenhandler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/builderhandler"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

// isBuilderRoute reports whether m is for the Assistant Builder or the Flow
// Builder (VOIP-1573). Both share this routing so the "route before the log
// entry exists" guarantee covers both.
//
// It compares the URI exactly (no query string, no suffix) and ignores the
// method. The method is ignored on purpose: a wrong method must be answered by
// processBuilder, not fall through to processRequest's default 404 branch,
// whose log line carries the whole request, body included. It is a plain string
// comparison, not a regular expression, because it runs for every request the
// service receives.
func isBuilderRoute(m *sock.Request) bool {
	return m.URI == builder.URIChat || m.URI == builder.URIStatus || m.URI == flowbuilder.URIChat
}

// processBuilder handles the Builder routes. processRequest calls it BEFORE it
// builds its own log entry, because that entry puts the whole request (body
// included) into a field and the tail of the function logs it again; the Builder
// body is the customer's own business description and must reach no log.
//
// CONTRACT, which design 4.7 depends on:
//
//   - m and m.Data are never put in any log or error text here;
//   - EVERY failure, including a body that does not unmarshal, is converted with
//     errorResponse and returned as (response, nil). Returning the error would
//     reach the queue consumer (consume.go), which logs err and publishes a
//     bare 500, losing the reason the client acts on;
//   - a panic becomes a bare 500 with no body: the panic value can carry the
//     input.
//
// It skips promReceivedRequestProcessTime; ai_manager_builder_chat_duration_seconds
// is the equivalent.
func (h *listenHandler) processBuilder(m *sock.Request) (response *sock.Response, err error) {
	// err is always nil on return, see the contract above.
	defer func() {
		if r := recover(); r != nil {
			// r is deliberately not logged or formatted.
			logrus.WithField("func", "processBuilder").Error("A builder request panicked.")
			// Each builder counts its own panics, so the Flow Builder never
			// shows up in the Assistant Builder's series.
			if m.URI == flowbuilder.URIChat {
				builderhandler.RecordFlowPanic()
			} else {
				builderhandler.RecordPanic()
			}
			response = simpleResponse(http.StatusInternalServerError)
			err = nil
		}
	}()

	ctx := context.Background()

	switch {
	case m.URI == builder.URIChat && m.Method == sock.RequestMethodPost:
		return h.processBuilderChatPost(ctx, m), nil

	case m.URI == flowbuilder.URIChat && m.Method == sock.RequestMethodPost:
		return h.processFlowBuilderChatPost(ctx, m), nil

	case m.URI == builder.URIStatus && m.Method == sock.RequestMethodGet:
		return h.processBuilderStatusGet(), nil

	default:
		// A Builder URI with the wrong method.
		return errorResponse(cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "the method is not allowed for this route")), nil
	}
}

func (h *listenHandler) processBuilderChatPost(ctx context.Context, m *sock.Request) *sock.Response {
	var req request.V1DataBuilderChatPost
	if errUnmarshal := json.Unmarshal(m.Data, &req); errUnmarshal != nil {
		// The error text is not used: json errors can quote the offending bytes.
		return errorResponse(cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "the request body could not be read"))
	}
	if req.CustomerID.IsNil() {
		return errorResponse(cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "the customer is missing"))
	}

	res, errChat := h.builderHandler.Chat(ctx, req.CustomerID, &builder.ChatRequest{
		Messages:     req.Messages,
		CurrentDraft: req.CurrentDraft,
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

func (h *listenHandler) processBuilderStatusGet() *sock.Response {
	data, errMarshal := json.Marshal(h.builderHandler.Status())
	if errMarshal != nil {
		return errorResponse(cerrors.Internal(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the response could not be written"))
	}

	return &sock.Response{StatusCode: http.StatusOK, DataType: "application/json", Data: data}
}
