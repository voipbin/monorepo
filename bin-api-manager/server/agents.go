package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	commonaddress "monorepo/bin-common-handler/models/address"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetAgents(c *gin.Context, params openapi_server.GetAgentsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAgents",
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

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}

	if params.TagIds != nil {
		filters["tag_ids"] = *params.TagIds
	}

	if params.Status != nil && string(*params.Status) != string(amagent.StatusNone) {
		filters["status"] = string(*params.Status)
	}

	tmps, err := h.serviceHandler.AgentGets(c.Request.Context(), &a, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) PostAgents(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAgents",
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

	var req openapi_server.PostAgentsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	username := ""
	if req.Username != nil {
		username = *req.Username
	}

	password := ""
	if req.Password != nil {
		password = *req.Password
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}

	ringMethod := amagent.RingMethodRingAll
	if req.RingMethod != nil {
		ringMethod = string(*req.RingMethod)
	}

	permission := amagent.PermissionNone
	if req.Permission != nil {
		permission = amagent.Permission(*req.Permission)
	}

	tagIDs := []uuid.UUID{}
	if req.TagIds != nil {
		for _, v := range *req.TagIds {
			tagIDs = append(tagIDs, uuid.FromStringOrNil(v))
		}
	}

	addresses, err := MarshalUnmarshal[[]commonaddress.Address](req.Addresses)
	if err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	res, err := h.serviceHandler.AgentCreate(c.Request.Context(), &a, username, password, name, detail, amagent.RingMethod(ringMethod), permission, tagIDs, addresses)
	if err != nil {
		log.Errorf("Could not create a flow for outoing call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetAgentsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAgentsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.AgentGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the agent info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteAgentsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteAgentsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.AgentDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the delete the agent info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAgentsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAgentsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutAgentsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}

	ringMethod := amagent.RingMethodRingAll
	if req.RingMethod != nil {
		ringMethod = string(*req.RingMethod)
	}

	// update the agent
	res, err := h.serviceHandler.AgentUpdate(c.Request.Context(), &a, target, name, detail, amagent.RingMethod(ringMethod))
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAgentsIdAddresses(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAgentsIdAddresses",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutAgentsIdAddressesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	addresses, err := MarshalUnmarshal[[]commonaddress.Address](req.Addresses)
	if err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	res, err := h.serviceHandler.AgentUpdateAddresses(c.Request.Context(), &a, target, addresses)
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAgentsIdTagIds(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAgentsIdTagIds",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutAgentsIdTagIdsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	tagIDs := []uuid.UUID{}
	if req.TagIds != nil {
		for _, v := range *req.TagIds {
			tagIDs = append(tagIDs, uuid.FromStringOrNil(v))
		}
	}

	res, err := h.serviceHandler.AgentUpdateTagIDs(c.Request.Context(), &a, target, tagIDs)
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAgentsIdStatus(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAgentsIdStatus",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutAgentsIdStatusJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	status := amagent.StatusNone
	if req.Status != nil {
		status = amagent.Status(*req.Status)
	}

	res, err := h.serviceHandler.AgentUpdateStatus(c.Request.Context(), &a, target, status)
	if err != nil {
		log.Errorf("Could not update the agent's status. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAgentsIdPermission(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAgentsIdPermission",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutAgentsIdPermissionJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	permission := amagent.PermissionNone
	if req.Permission != nil {
		permission = amagent.Permission(*req.Permission)
	}

	res, err := h.serviceHandler.AgentUpdatePermission(c.Request.Context(), &a, target, permission)
	if err != nil {
		log.Errorf("Could not update the agent's permission. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAgentsIdPassword(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAgentsIdPassword",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutAgentsIdPasswordJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	password := ""
	if req.Password != nil {
		password = *req.Password
	}
	if password == "" {
		log.Error("Empty password is not valid.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.AgentUpdatePassword(c.Request.Context(), &a, target, password)
	if err != nil {
		log.Errorf("Could not update the agent's password. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
