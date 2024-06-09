package agentresources

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// agentResourcesGET handles GET /agent_resources request.
// It returns list of agent resources of the given user.
//
//	@Summary		List of agent resources
//	@Description	get agent resources of the user
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Param			reference_type		query		string	false	"reference type"
//	@Param			status		query		string	false	"Agent status"
//	@Success		200			{object}	response.BodyAgentResourcesGET
//	@Router			/v1.0/agent_resources [get]
func agentResourcesGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "agentResourcesGET",
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

	var req request.ParamAgentResourcesGET
	if err := c.BindQuery(&req); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	} else if pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to max. page_size: %d", pageSize)
	}

	// filters
	filters := map[string]string{
		"customer_id":    a.CustomerID.String(),
		"owner_id":       a.ID.String(),
		"reference_type": req.ReferenceType,
		"deleted":        "false",
	}

	// get tmps
	tmps, err := serviceHandler.AgentResourceGets(c.Request.Context(), &a, pageSize, req.PageToken, filters)
	if err != nil {
		logrus.Errorf("Could not get agents info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyAgentResourcesGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// agentResourcesIDGet handles GET /agent_resources/<resource-id> request.
// It gets the agent resource.
//
//	@Summary		Get the agent resource
//	@Description	Get the agent resource of the given id
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the agent resource to retrieve"
//	@Success		200
//	@Router			/v1.0/agent_resources/{id} [get]
func agentResourcesIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "agentResourcesIDGet",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get
	res, err := serviceHandler.AgentResourceGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Infof("Could not get the agent resource info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// agentResourcesIDDelete handles DELETE /agent_resources/<resource-id> request.
// It deletes the agent resource.
//
//	@Summary		Delete the agent resource
//	@Description	Delete the agent resource of the given id
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the agent resource to delete"
//	@Success		200
//	@Router			/v1.0/agent_resources/{id} [delete]
func agentResourcesIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "agentResourcesIDDelete",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	if id == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// delete
	res, err := serviceHandler.AgentResourceDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Infof("Could not get the delete the agent resource info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
