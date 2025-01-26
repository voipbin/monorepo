package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetMe(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetMe",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent":    a,
		"username": a.Username,
	})

	res, err := h.serviceHandler.AgentGet(c.Request.Context(), &a, a.ID)
	if err != nil {
		log.Infof("Could not get the agent info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
