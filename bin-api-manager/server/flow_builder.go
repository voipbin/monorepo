package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// flowBuilderMaxBodyBytes caps the request body. It is NOT the Assistant
// Builder's 160 KiB: a flow draft rides along on every turn (up to 60
// actions of up to 4 KiB of option each, plus positions and labels) next to a
// conversation of up to 40000 characters, about 380 KiB at the limits
// (design doc 5). The initial value is not measured. It must stay below the
// body limit of whatever sits in front of api-manager.
const flowBuilderMaxBodyBytes = 512 << 10

// PostFlowBuilderChat runs one turn of the flow builder conversation.
//
// Like PostAiBuilderChat, it checks only that the caller is a logged-in Agent,
// and does it BEFORE reading the body. Who may use it is decided in
// servicehandler.FlowBuilderChat. Nothing the customer wrote is logged: not
// the body, not the draft, not a parse error, not the error text of a failed
// call.
func (h *server) PostFlowBuilderChat(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "PostFlowBuilderChat",
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Error("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent_id":    a.AgentID(),
		"customer_id": a.CustomerID,
	})

	if !a.IsAgent() {
		log.Info("The flow builder is for logged-in agents only.")
		abortWithError(c, cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "PERMISSION_DENIED", "You do not have permission to access this resource."))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, flowBuilderMaxBodyBytes)

	var req openapi_server.PostFlowBuilderChatJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		// err is not logged or attached: encoding/json errors can quote the
		// bytes they stopped at, and this is the customer's own text.
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			log.Info("The flow builder request body was too large.")
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, builder.ReasonInputTooLarge, "The request body is too large."))
			return
		}
		log.Info("Could not parse the flow builder request.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	domainReq, ok := toFlowBuilderChatRequest(&req)
	if !ok {
		log.Info("Could not read the flow builder request.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	res, err := h.serviceHandler.FlowBuilderChat(c.Request.Context(), a, domainReq)
	if err != nil {
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// toFlowBuilderChatRequest copies the generated request type into the domain
// type the services use. The two have the same JSON shape (the OpenAPI schema
// is the wire contract of the domain type), so a JSON round trip is the copy:
// it keeps every key of an action's open-ended `option` and needs no
// per-action code. It returns false if the round trip fails, which it cannot
// for a value that was just decoded from JSON.
func toFlowBuilderChatRequest(req *openapi_server.PostFlowBuilderChatJSONRequestBody) (*flowbuilder.ChatRequest, bool) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, false
	}
	var out flowbuilder.ChatRequest
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, false
	}
	return &out, true
}
