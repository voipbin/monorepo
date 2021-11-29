package agents

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// agentsPOST handles POST /agents request.
// It creates a new agent.
// @Summary Create a new agent.
// @Description create a new agent
// @Produce  json
// @Param agent body request.BodyAgentsPOST true "The agent detail"
// @Success 200 {object} agent.Agent
// @Router /v1.0/agents [post]
func agentsPOST(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "agentsPOST",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	var req request.BodyAgentsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create
	res, err := serviceHandler.AgentCreate(&u, req.Username, req.Password, req.Name, req.Detail, req.RingMethod, uint64(req.Permission), req.TagIDs, req.Addresses)
	if err != nil {
		log.Errorf("Could not create a flow for outoing call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// agentIDDelete handles DELETE /agents/<agent-id> request.
// It deletes the agent.
// @Summary Delete the agent
// @Description Delete the agent of the given id
// @Produce json
// @Param id path string true "The ID of the agent"
// @Success 200
// @Router /v1.0/agents/{id} [delete]
func agentsIDDelete(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "agentsIDDelete",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

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
	err := serviceHandler.AgentDelete(&u, id)
	if err != nil {
		log.Infof("Could not get the delete the agent info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// agentIDGet handles GET /agents/<agent-id> request.
// It gets the agent.
// @Summary Get the agent
// @Description Get the agent of the given id
// @Produce json
// @Param id path string true "The ID of the agent"
// @Success 200
// @Router /v1.0/agents/{id} [get]
func agentsIDGet(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "agentsIDGet",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get
	res, err := serviceHandler.AgentGet(&u, id)
	if err != nil {
		log.Infof("Could not get the agent info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// agentsGET handles GET /agents request.
// It returns list of agents of the given user.
// @Summary List agents
// @Description get agents of the user
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Param tag_ids query string false "Comma seperated tag ids"
// @Param status query string false "Agent status"
// @Success 200 {object} response.BodyAgentsGET
// @Router /v1.0/agents [get]
func agentsGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "agentsGET",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	var req request.ParamAgentsGET
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

	tagIDs := []uuid.UUID{}
	if req.TagIDs != "" {
		tags := strings.Split(req.TagIDs, ",")
		for _, tag := range tags {
			t := uuid.FromStringOrNil(tag)
			tagIDs = append(tagIDs, t)
		}
	}

	// get tmps
	tmps, err := serviceHandler.AgentGets(&u, pageSize, req.PageToken, tagIDs, agent.Status(req.Status))
	if err != nil {
		logrus.Errorf("Could not get agents info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyAgentsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// agentsIDPUT handles PUT /agents/{id} request.
// It updates a agent basic info with the given info.
// And returns updated agent info if it succeed.
// @Summary Update an agent and reuturns updated agent info.
// @Description Update an agent and returns detail updated agent info.
// @Produce json
// @Param id path string true "The ID of the agent"
// @Param name body string true "Agent's name"
// @Param detail body string true "Agent's detail"
// @Param ring_method body string true "Agent's ring method"
// @Success 200
// @Router /v1.0/agents/{id} [put]
func agentsIDPUT(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "agentsIDPUT",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	var req request.BodyAgentsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the agent
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.AgentUpdate(&u, id, req.Name, req.Detail, agent.RingMethod(req.RingMethod)); err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// agentsIDAddressesPUT handles PUT /agents/{id}/Addresses request.
// It updates a agent's addresses info with the given info.
// And returns updated agent info if it succeed.
// @Summary Update an agent info.
// @Description Update an agent addresses info.
// @Produce json
// @Param id path string true "The ID of the agent"
// @Param addresses body [{object}] address.Address true "Agent's addresses"
// @Success 200
// @Router /v1.0/agents/{id}/addresses [put]
func agentsIDAddressesPUT(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "agentsIDAddressesPUT",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	var req request.BodyAgentsIDAddressesPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the agent
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.AgentUpdateAddresses(&u, id, req.Addresses); err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// agentsIDTagIDsPUT handles PUT /agents/{id}/tag_ids request.
// It updates a agent's tag_ids info with the given info.
// @Summary Update an agent's tag_id info.
// @Description Update an agent tag_ids info.
// @Produce json
// @Param id path string true "The ID of the agent"
// @Param addresses body [string] true "Agent's tag ids"
// @Success 200
// @Router /v1.0/agents/{id}/tag_ids [put]
func agentsIDTagIDsPUT(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "agentsIDTagIDsPUT",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	var req request.BodyAgentsIDAddressesPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the agent
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.AgentUpdateAddresses(&u, id, req.Addresses); err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
