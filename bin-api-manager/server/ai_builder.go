package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// builderMaxBodyBytes caps the request body. The conversation limit is 40000
// characters; Korean is 3 bytes a character in UTF-8 and JSON escaping adds
// more, so 160 KiB leaves room for a conversation at the limit without letting a
// caller make the server read an unbounded body.
const builderMaxBodyBytes = 160 << 10

// PostAiBuilderChat runs one turn of the assistant builder conversation.
//
// The handler checks only that the caller is a logged-in Agent, and does it
// BEFORE reading the body. Who may use the builder (the admin or manager
// permission) is decided in servicehandler.AIBuilderChat, as the other handlers
// do.
//
// Nothing the customer wrote is logged: not the body, not a parse error (which
// can quote it), not the error text of a failed call.
func (h *server) PostAiBuilderChat(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "PostAiBuilderChat",
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
		log.Info("The builder is for logged-in agents only.")
		abortWithError(c, cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "PERMISSION_DENIED", "You do not have permission to access this resource."))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, builderMaxBodyBytes)

	var req openapi_server.PostAiBuilderChatJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		// err is not logged or attached: encoding/json errors can quote the
		// bytes they stopped at, and this is the customer's own text.
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			log.Info("The builder request body was too large.")
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, builder.ReasonInputTooLarge, "The request body is too large."))
			return
		}
		log.Info("Could not parse the builder request.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	res, err := h.serviceHandler.AIBuilderChat(c.Request.Context(), a, toBuilderChatRequest(&req))
	if err != nil {
		abortWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetAiBuilderStatus reports whether the assistant builder is available.
//
// It answers 200 with available=false for a caller who is not an Agent, and for
// any failure to find out, so the client can hide the entry point without
// handling an error. A missing identity is still 401: that is a login problem,
// not a status.
func (h *server) GetAiBuilderStatus(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "GetAiBuilderStatus",
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Error("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	if !a.IsAgent() {
		c.JSON(http.StatusOK, &builder.StatusResponse{Available: false})
		return
	}

	res, err := h.serviceHandler.AIBuilderStatus(c.Request.Context(), a)
	if err != nil || res == nil {
		log.WithField("agent_id", a.AgentID()).Info("Could not get the builder status.")
		c.JSON(http.StatusOK, &builder.StatusResponse{Available: false})
		return
	}

	c.JSON(http.StatusOK, res)
}

// toBuilderChatRequest copies the generated request type into the domain type
// the services use. The two are separate on purpose: the generated one follows
// the OpenAPI spec, the domain one is what ai-manager validates and receives.
func toBuilderChatRequest(req *openapi_server.PostAiBuilderChatJSONRequestBody) *builder.ChatRequest {
	out := &builder.ChatRequest{
		Messages: make([]builder.Message, 0, len(req.Messages)),
	}
	for _, m := range req.Messages {
		out.Messages = append(out.Messages, builder.Message{Role: string(m.Role), Content: m.Content})
	}
	if req.CurrentDraft != nil {
		out.CurrentDraft = &builder.Draft{
			Name:       req.CurrentDraft.Name,
			Detail:     req.CurrentDraft.Detail,
			InitPrompt: req.CurrentDraft.InitPrompt,
			ToolNames:  toolNamesToStrings(req.CurrentDraft.ToolNames),
		}
	}
	return out
}

func toolNamesToStrings(names []openapi_server.AIManagerToolName) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, string(n))
	}
	return out
}
