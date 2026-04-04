package server

import (
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
		c.AbortWithStatus(400)
		return
	}
	log = log.WithField("agent", a)

	res, err := h.serviceHandler.AgentGet(c.Request.Context(), a, a.AgentID())
	if err != nil {
		log.Infof("Could not get the agent info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
