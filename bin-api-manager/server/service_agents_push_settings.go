package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	pmnotification "monorepo/bin-push-manager/models/notification"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// GetServiceAgentsMePushSettings handles GET /service_agents/me/push_settings.
func (h *server) GetServiceAgentsMePushSettings(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsMePushSettings",
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

	res, err := h.serviceHandler.ServiceAgentPushSettingsGet(c.Request.Context(), a)
	if err != nil {
		log.Errorf("Could not get the push settings. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PutServiceAgentsMePushSettings handles PUT /service_agents/me/push_settings.
func (h *server) PutServiceAgentsMePushSettings(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsMePushSettings",
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

	var req openapi_server.PutServiceAgentsMePushSettingsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
		return
	}
	if req.Kinds == nil {
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The kinds are required."))
		return
	}

	kinds := make(map[pmnotification.Kind]bool, len(req.Kinds))
	for k, v := range req.Kinds {
		kinds[pmnotification.Kind(k)] = v
	}

	res, err := h.serviceHandler.ServiceAgentPushSettingsUpdate(c.Request.Context(), a, kinds)
	if err != nil {
		log.Errorf("Could not update the push settings. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PutServiceAgentsMePushMutesChatId handles PUT /service_agents/me/push_mutes/{chat_id}.
func (h *server) PutServiceAgentsMePushMutesChatId(c *gin.Context, chatID openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsMePushMutesChatId",
		"request_address": c.ClientIP,
		"chat_id":         chatID,
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

	res, err := h.serviceHandler.ServiceAgentPushMuteAdd(c.Request.Context(), a, uuid.UUID(chatID))
	if err != nil {
		log.Errorf("Could not mute the chat. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteServiceAgentsMePushMutesChatId handles DELETE /service_agents/me/push_mutes/{chat_id}.
func (h *server) DeleteServiceAgentsMePushMutesChatId(c *gin.Context, chatID openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsMePushMutesChatId",
		"request_address": c.ClientIP,
		"chat_id":         chatID,
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

	res, err := h.serviceHandler.ServiceAgentPushMuteRemove(c.Request.Context(), a, uuid.UUID(chatID))
	if err != nil {
		log.Errorf("Could not unmute the chat. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
