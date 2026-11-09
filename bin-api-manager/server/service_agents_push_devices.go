package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	pmdevice "monorepo/bin-push-manager/models/device"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// PostServiceAgentsPushDevices handles POST /service_agents/push_devices.
// The request carries a push token, so it is never logged.
func (h *server) PostServiceAgentsPushDevices(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsPushDevices",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req openapi_server.PostServiceAgentsPushDevicesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}

	res, err := h.serviceHandler.ServiceAgentPushDeviceCreate(c.Request.Context(), a, pmdevice.Platform(req.Platform), req.Token)
	if err != nil {
		log.Errorf("Could not register the push device. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteServiceAgentsPushDevicesId handles DELETE /service_agents/push_devices/{id}.
func (h *server) DeleteServiceAgentsPushDevicesId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsPushDevicesId",
		"request_address": c.ClientIP,
		"device_id":       id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	res, err := h.serviceHandler.ServiceAgentPushDeviceDelete(c.Request.Context(), a, uuid.UUID(id))
	if err != nil {
		log.Errorf("Could not delete the push device. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
