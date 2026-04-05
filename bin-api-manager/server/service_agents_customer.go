package server

import (

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) GetServiceAgentsCustomer(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsCustomer",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		c.AbortWithStatus(400)
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	res, err := h.serviceHandler.ServiceAgentCustomerGet(c.Request.Context(), a)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
