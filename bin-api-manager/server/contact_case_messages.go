package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// PostContactCasesIdMessages handles POST /contact_cases/{id}/messages (design §4.5).
// customer_id is derived only from the authenticated caller's context
// (getAuthIdentity) -- never from source/destination/case_id supplied in
// the request, which are validated server-side by CaseMessageSend's
// 6-step sequence (case ownership, destination-to-case binding,
// source-ownership, conversation resolution, fail-open metadata write,
// send).
func (h *server) PostContactCasesIdMessages(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactCasesIdMessages",
		"request_address": c.ClientIP(),
		"id":              id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	var req openapi_server.PostContactCasesIdMessagesJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	caseID := uuid.UUID(id)

	res, err := h.serviceHandler.CaseMessageSend(c.Request.Context(), a, caseID, req.Source, req.Destination, req.Text)
	if err != nil {
		log.Errorf("Could not send the case message. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
