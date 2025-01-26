package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostOutplans(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostOutplans",
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

	var req openapi_server.PostOutplansJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	source := ConvertCommonAddress(req.Source)

	res, err := h.serviceHandler.OutplanCreate(c.Request.Context(), &a, req.Name, req.Detail, &source, req.DialTimeout, req.TryInterval, req.MaxTryCount0, req.MaxTryCount1, req.MaxTryCount2, req.MaxTryCount3, req.MaxTryCount4)
	if err != nil {
		log.Errorf("Could not create a outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetOutplans(c *gin.Context, params openapi_server.GetOutplansParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutplans",
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

	tmps, err := h.serviceHandler.OutplanGetsByCustomerID(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a outplan list. err: %v", err)
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

func (h *server) GetOutplansId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutplansId",
		"request_address": c.ClientIP,
		"outplan_id":      id,
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

	res, err := h.serviceHandler.OutplanGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteOutplansId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteOutplansId",
		"request_address": c.ClientIP,
		"outplan_id":      id,
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

	res, err := h.serviceHandler.OutplanDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutOutplansId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outplansIDPUT",
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

	var req openapi_server.PutOutplansIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.OutplanUpdateBasicInfo(c.Request.Context(), &a, target, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutOutplansIdDialInfo(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutOutplansIdDialInfo",
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

	var req openapi_server.PutOutplansIdDialInfoJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	source := ConvertCommonAddress(req.Source)

	res, err := h.serviceHandler.OutplanUpdateDialInfo(c.Request.Context(), &a, target, &source, req.DialTimeout, req.TryInterval, req.MaxTryCount0, req.MaxTryCount1, req.MaxTryCount2, req.MaxTryCount3, req.MaxTryCount4)
	if err != nil {
		log.Errorf("Could not update the outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
