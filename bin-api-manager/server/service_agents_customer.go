package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetServiceAgentsCustomer(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsCustomer",
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
		"agent": a,
	})

	res, err := h.serviceHandler.ServiceAgentCustomerGet(c.Request.Context(), &a)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
