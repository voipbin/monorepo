package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"

	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetAccesskeys(c *gin.Context, params openapi_server.GetAccesskeysParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAccesskeys",
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

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.AccesskeyGets(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get calls info. err: %v", err)
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

func (h *server) PostAccesskeys(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAccesskeys",
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

	var req openapi_server.PostAccesskeysJSONBody
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

	expire := 0
	if req.Expire != nil {
		expire = int(*req.Expire)
	}

	res, err := h.serviceHandler.AccesskeyCreate(c.Request.Context(), &a, name, detail, int32(expire))
	if err != nil {
		log.Errorf("Could not create a accesskey. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetAccesskeysId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAccesskeysId",
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
	log = log.WithField("accesskey_id", target)

	res, err := h.serviceHandler.AccesskeyGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteAccesskeysId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteAccesskeysId",
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
	log = log.WithField("accesskey_id", target)

	res, err := h.serviceHandler.AccesskeyDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAccesskeysId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAccesskeysId",
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
	log = log.WithField("accesskey_id", target)

	var req openapi_server.PutAccesskeysIdJSONRequestBody
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

	res, err := h.serviceHandler.AccesskeyUpdate(c.Request.Context(), &a, target, name, detail)
	if err != nil {
		log.Errorf("Could not update the accesskey. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
