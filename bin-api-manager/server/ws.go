package server

import (

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetWs(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetWs",
		"request_address": c.ClientIP,
	})
	log.Debugf("Received websocket request.")

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	if err := h.serviceHandler.WebsockCreate(c.Request.Context(), a, c.Writer, c.Request); err != nil {
		log.Errorf("Could not handler the websocket correctly. err: %v", err)
	}
}
