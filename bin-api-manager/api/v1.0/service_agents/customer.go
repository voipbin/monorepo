package service_agents

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// customerGET handles GET /service_agents/customer request.
// It returns detail customer info.
//
//	@Summary		Get detail call info.
//	@Description	Returns detail customer info of the authenticated agent.
//	@Produce		json
//	@Success		200	{object}	customer.Customer
//	@Router			/v1.0/service_agents/customer [get]
func customerGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "customerGET",
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

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentCustomerGet(c.Request.Context(), &a)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
