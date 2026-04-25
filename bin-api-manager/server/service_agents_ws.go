package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetServiceAgentsWs(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsWs",
		"request_address": c.ClientIP,
	})
	log.Debugf("Received websocket request.")

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	if err := h.serviceHandler.WebsockCreate(c.Request.Context(), a, c.Writer, c.Request); err != nil {
		log.Errorf("Could not handler the websocket correctly. err: %v", err)
	}
}
