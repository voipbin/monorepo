package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	fmaction "monorepo/bin-flow-manager/models/action"

	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetActiveflows(c *gin.Context, params openapi_server.GetActiveflowsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "activeflowsGET",
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

	tmps, err := h.serviceHandler.ActiveflowGets(c.Request.Context(), &a, pageSize, pageToken)
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

func (h *server) PostActiveflows(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostActiveflows",
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

	var req openapi_server.PostActiveflowsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	id := uuid.Nil
	if req.Id != nil {
		id = uuid.FromStringOrNil(*req.Id)
	}

	flowID := uuid.Nil
	if req.FlowId != nil {
		flowID = uuid.FromStringOrNil(*req.FlowId)
	}

	actions := []fmaction.Action{}
	if req.Actions != nil {
		for _, v := range *req.Actions {
			actions = append(actions, ConvertFlowManagerAction(v))
		}
	}

	res, err := h.serviceHandler.ActiveflowCreate(c.Request.Context(), &a, id, flowID, actions)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetActiveflowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetActiveflowsId",
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
	log = log.WithField("activeflow_id", target)

	res, err := h.serviceHandler.ActiveflowGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a activeflow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteActiveflowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteActiveflowsId",
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
	log = log.WithField("activeflow_id", target)

	res, err := h.serviceHandler.ActiveflowDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the activeflow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostActiveflowsIdStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostActiveflowsIdStop",
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
	log = log.WithField("activeflow_id", target)

	res, err := h.serviceHandler.ActiveflowStop(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not stop the activeflow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
