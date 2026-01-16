package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	qmqueue "monorepo/bin-queue-manager/models/queue"

	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetQueues(c *gin.Context, params openapi_server.GetQueuesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetQueues",
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

	tmps, err := h.serviceHandler.QueueList(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get queues info. err: %v", err)
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

func (h *server) PostQueues(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostQueues",
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

	var req openapi_server.PostQueuesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	tagIDs := []uuid.UUID{}
	for _, v := range req.TagIds {
		tagIDs = append(tagIDs, uuid.FromStringOrNil(v))
	}

	waitFlowID := uuid.FromStringOrNil(req.WaitFlowId)

	res, err := h.serviceHandler.QueueCreate(
		c.Request.Context(),
		&a,
		req.Name,
		req.Detail,
		qmqueue.RoutingMethod(req.RoutingMethod),
		tagIDs,
		waitFlowID,
		req.WaitTimeout,
		req.ServiceTimeout,
	)
	if err != nil {
		log.Errorf("Could not create a queue. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteQueuesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteQueuesId",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	res, err := h.serviceHandler.QueueDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the delete the queue info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetQueuesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetQueuesId",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	res, err := h.serviceHandler.QueueGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Infof("Could not get the queue info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutQueuesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutQueuesId",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	var req openapi_server.PutQueuesIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	tagIDs := []uuid.UUID{}
	for _, v := range req.TagIds {
		tagIDs = append(tagIDs, uuid.FromStringOrNil(v))
	}

	waitFlowID := uuid.FromStringOrNil(req.WaitFlowId)

	res, err := h.serviceHandler.QueueUpdate(c.Request.Context(), &a, target, req.Name, req.Detail, qmqueue.RoutingMethod(req.RoutingMethod), tagIDs, waitFlowID, req.WaitTimeout, req.ServiceTimeout)
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutQueuesIdTagIds(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutQueuesIdTagIds",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	var req openapi_server.PutQueuesIdTagIdsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	tagIDs := []uuid.UUID{}
	for _, v := range req.TagIds {
		tagIDs = append(tagIDs, uuid.FromStringOrNil(v))
	}

	res, err := h.serviceHandler.QueueUpdateTagIDs(c.Request.Context(), &a, target, tagIDs)
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutQueuesIdRoutingMethod(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutQueuesIdRoutingMethod",
		"request_address": c.ClientIP,
		"queue_id":        id,
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

	var req openapi_server.PutQueuesIdRoutingMethodJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.QueueUpdateRoutingMethod(c.Request.Context(), &a, target, qmqueue.RoutingMethod(req.RoutingMethod))
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
