package service_agents

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// extensionsGET handles GET service_agents/extensions request.
// It returns list of extensions of the given agent.

// @Summary		Get list of extensions
// @Description	get extensions of the agent
// @Produce		json
// @Success		200			{object}	response.BodyServiceAgentsExtensionsGET
// @Router			/v1.0/service_agents/extensions [get]
func extensionsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "extensionsGET",
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

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get
	tmps, err := serviceHandler.ExtensionGetsByOwner(c.Request.Context(), &a)
	if err != nil {
		log.Errorf("Could not get extensions info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyServiceAgentsExtensionsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// extensionsIDGET handles GET service_agents/extensions/<extension-id> request.
// It returns details of extension of the given id.

// @Summary		Get details of extension
// @Description	get extension detail info
// @Produce		json
// @Success		200			{object}	extension.extension
// @Router			/v1.0/service_agents/extensions/{id} [get]
func extensionsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "extensionsIDGET",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("extension_id", id)
	log.Debug("Executing extensionsIDGET.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get
	res, err := serviceHandler.ExtensionGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
