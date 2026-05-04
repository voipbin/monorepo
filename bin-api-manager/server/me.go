package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetMe(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetMe",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	if !a.IsAgent() {
		log.Infof("Non-agent auth type attempted GET /me. type: %s", a.Type)
		abortWithError(c, cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "AGENT_AUTH_REQUIRED", "This endpoint requires agent authentication."))
		return
	}

	res, err := h.serviceHandler.AgentGet(c.Request.Context(), a, a.AgentID())
	if err != nil {
		log.Infof("Could not get the agent info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
