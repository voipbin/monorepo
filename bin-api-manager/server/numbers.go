package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetNumbers(c *gin.Context, params openapi_server.GetNumbersParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetNumbers",
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

	tmps, err := h.serviceHandler.NumberList(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a order number list. err: %v", err)
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

func (h *server) GetNumbersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetNumbersId",
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

	res, err := h.serviceHandler.NumberGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get an order number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostNumbers(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostNumbers",
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

	var req openapi_server.PostNumbersJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	callFlowID := uuid.FromStringOrNil(req.CallFlowId)
	messageFlowID := uuid.FromStringOrNil(req.MessageFlowId)

	numb, err := h.serviceHandler.NumberCreate(c.Request.Context(), &a, req.Number, callFlowID, messageFlowID, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create the number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, numb)
}

func (h *server) DeleteNumbersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteNumbersId",
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

	res, err := h.serviceHandler.NumberDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete an order number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutNumbersId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutNumbersId",
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

	var req openapi_server.PutNumbersIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	callFlowID := uuid.FromStringOrNil(req.CallFlowId)
	messageFlowID := uuid.FromStringOrNil(req.MessageFlowId)

	res, err := h.serviceHandler.NumberUpdate(c.Request.Context(), &a, target, callFlowID, messageFlowID, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update a number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutNumbersIdFlowIds(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutNumbersIdFlowIds",
		"request_address": c.ClientIP,
		"number_id":       id,
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

	var req openapi_server.PutNumbersIdFlowIdsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	callFlowID := uuid.FromStringOrNil(req.CallFlowId)
	messageFlowID := uuid.FromStringOrNil(req.MessageFlowId)

	res, err := h.serviceHandler.NumberUpdateFlowIDs(c.Request.Context(), &a, target, callFlowID, messageFlowID)
	if err != nil {
		log.Errorf("Could not update a number. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostNumbersRenew(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostNumbersRenew",
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

	var req openapi_server.PostNumbersRenewJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.NumberRenew(c.Request.Context(), &a, req.TmRenew)
	if err != nil {
		log.Errorf("Could not renew the numbers. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
