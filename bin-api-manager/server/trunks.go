package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostTrunks(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostTrunks",
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

	var req openapi_server.PostTrunksJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	authTyps := []rmsipauth.AuthType{}
	for _, v := range req.AuthTypes {
		authTyps = append(authTyps, rmsipauth.AuthType(v))
	}

	res, err := h.serviceHandler.TrunkCreate(c.Request.Context(), &a, req.Name, req.Detail, req.DomainName, authTyps, req.Username, req.Password, req.AllowedIps)
	if err != nil {
		log.Errorf("Could not create a trunk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetTrunks(c *gin.Context, params openapi_server.GetTrunksParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTrunks",
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

	tmps, err := h.serviceHandler.TrunkGets(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a trunk list. err: %v", err)
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

func (h *server) GetTrunksId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTrunksId",
		"request_address": c.ClientIP,
		"trunk_id":        id,
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

	res, err := h.serviceHandler.TrunkGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a trunk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutTrunksId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutTrunksId",
		"request_address": c.ClientIP,
		"trunk_id":        id,
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

	var req openapi_server.PutTrunksIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	authTyps := []rmsipauth.AuthType{}
	for _, v := range req.AuthTypes {
		authTyps = append(authTyps, rmsipauth.AuthType(v))
	}

	res, err := h.serviceHandler.TrunkUpdateBasicInfo(c.Request.Context(), &a, target, req.Name, req.Detail, authTyps, req.Username, req.Password, req.AllowedIps)
	if err != nil {
		log.Errorf("Could not update the trunk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteTrunksId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteTrunksId",
		"request_address": c.ClientIP,
		"trunk_id":        id,
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

	res, err := h.serviceHandler.TrunkDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the trunk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
