package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// PostServiceAgentsCallsIdParticipants adds a colleague's extension to a call the calling agent owns.
// Thin by design: parse the body, one service-handler call. Only the address type and target are taken from the
// request, the service handler builds the address that is passed on.
func (h *server) PostServiceAgentsCallsIdParticipants(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsCallsIdParticipants",
		"request_address": c.ClientIP(),
		"call_id":         id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	callID, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid call ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	var req openapi_server.PostServiceAgentsCallsIdParticipantsJSONBody
	if errBind := c.BindJSON(&req); errBind != nil {
		log.Errorf("Could not parse the request. err: %v", errBind)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(errBind))
		return
	}

	res, err := h.serviceHandler.ServiceAgentCallParticipantAdd(c.Request.Context(), a, callID, commonaddress.Address{
		Type:   commonaddress.Type(req.Address.Type),
		Target: req.Address.Target,
	})
	if err != nil {
		log.Errorf("Could not add the participant. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
