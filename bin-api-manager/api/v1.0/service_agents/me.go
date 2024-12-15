package service_agents

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// meGET handles the GET request for /service_agents/me.
// It returns the authenticated agent's information.
//
// @Summary      Get detailed agent information
// @Description  Retrieve the authenticated agent's information.
// @Produce      json
// @Success      200        {object} agent.Agent
// @Router       /v1.0/service_agents/me [get]
func meGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "meGET",
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

	res, err := serviceHandler.ServiceAgentMeGet(c.Request.Context(), &a)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// mePUT handles the PUT request for /service_agents/me
// It updates the authenticated agent's basic information with the provided details.
// Returns the updated agent's information upon success.
//
// @Summary      Update an agent's basic information
// @Description  Updates the authenticated agent's details and returns the updated information.
// @Produce      json
// @Param        id          path    string                      true  "The ID of the agent"
// @Param        update_info body    request.BodyServiceAgentsMePUT true  "Agent's updated details"
// @Success      200        {object} agent.Agent
// @Router       /v1.0/service_agents/me [put]
func mePUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "mePUT",
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

	var req request.BodyServiceAgentsMePUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentMeUpdate(c.Request.Context(), &a, req.Name, req.Detail, req.RingMethod)
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// meAddressesPUT handles the PUT request for /service_agents/me/addresses.
// It updates the authenticated agent's address information with the provided details.
// Returns the updated address information upon success.
//
// @Summary      Update an agent's address information
// @Description  Updates the authenticated agent's address details and returns the updated information.
// @Produce      json
// @Param        id          path    string                      true  "The ID of the agent"
// @Param        update_info body    request.BodyServiceAgentsMePUT true  "Agent's updated address details"
// @Success      200        {object} agent.Agent
// @Router       /v1.0/service_agents/me/addresses [put]
func meAddressesPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "meAddressesPUT",
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

	var req request.BodyServiceAgentsMeAddressesPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentMeUpdateAddresses(c.Request.Context(), &a, req.Addresses)
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// meStatusPUT handles the PUT request for /service_agents/me/status.
// It updates the authenticated agent's status information with the provided details.
//
// @Summary      Update an agent's status information
// @Description  Updates the authenticated agent's status details and returns the updated information.
// @Produce      json
// @Param        id          path    string                      true  "The ID of the agent"
// @Param        update_info body    request.BodyServiceAgentsMeStatusPUT true  "Agent's updated status details"
// @Success      200        {object} agent.Agent
// @Router       /v1.0/service_agents/me/status [put]
func meStatusPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "meStatusPUT",
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

	var req request.BodyServiceAgentsMeStatusPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the agent
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentMeUpdateStatus(c.Request.Context(), &a, req.Status)
	if err != nil {
		log.Errorf("Could not update the agent's status. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// meStatusPUT handles the PUT request for /service_agents/me/password.
// It updates the authenticated agent's password information with the provided details.
//
// @Summary      Update an agent's password information
// @Description  Updates the authenticated agent's password details and returns the updated information.
// @Produce      json
// @Param        id          path    string                      true  "The ID of the agent"
// @Param        update_info body    request.BodyServiceAgentsMeStatusPUT true  "Agent's updated password"
// @Success      200        {object} agent.Agent
// @Router       /v1.0/service_agents/me/password [put]
func mePasswordPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "mePasswordPUT",
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

	var req request.BodyServiceAgentsMePasswordPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the agent
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentMeUpdatePassword(c.Request.Context(), &a, req.Password)
	if err != nil {
		log.Errorf("Could not update the agent's password. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
