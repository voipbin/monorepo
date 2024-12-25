package me

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// meGET handles GET /me request.
// It gets the agent.
//
//	@Summary		Get the logged in agent
//	@Description	Get the logged in agent information
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the agent"
//	@Success		200
//	@Router			/v1.0/me [get]
func meGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "meGet",
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

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get
	res, err := serviceHandler.AgentGet(c.Request.Context(), &a, a.ID)
	if err != nil {
		log.Infof("Could not get the agent info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
