package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// GetServiceAgentsCallsIdQueuecall returns the queue entry of a call the
// calling agent owns. Thin by design: auth identity, one service-handler call.
func (h *server) GetServiceAgentsCallsIdQueuecall(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsCallsIdQueuecall",
		"request_address": c.ClientIP,
		"call_id":         id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid call ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentCallQueuecallGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get the queuecall of the call. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
