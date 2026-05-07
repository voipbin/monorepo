package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GetOutboundConfig handles GET /outbound_config
// Returns the outbound config for the authenticated customer (self-service, no ID required).
func (h *server) GetOutboundConfig(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutboundConfig",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("auth", a)

	res, err := h.serviceHandler.OutboundConfigSelfGet(c.Request.Context(), a)
	if err != nil {
		log.Infof("Could not get the outbound config info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PutOutboundConfig handles PUT /outbound_config
// Updates the outbound config for the authenticated customer (self-service, no ID required).
func (h *server) PutOutboundConfig(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutOutboundConfig",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("auth", a)

	var req openapi_server.PutOutboundConfigJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_JSON_BODY",
			"The request body is not valid JSON.",
		))
		return
	}

	updateReq := convertOutboundConfigUpdateRequest(req)

	res, err := h.serviceHandler.OutboundConfigSelfUpdate(c.Request.Context(), a, updateReq)
	if err != nil {
		log.Errorf("Could not update outbound config. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
