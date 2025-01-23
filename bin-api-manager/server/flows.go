package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostFlows(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostFlows",
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

	var req openapi_server.PostFlowsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	actions := []fmaction.Action{}
	for _, v := range req.Actions {
		actions = append(actions, ConvertFlowManagerAction(v))
	}

	res, err := h.serviceHandler.FlowCreate(c.Request.Context(), &a, req.Name, req.Detail, actions, true)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetFlows(c *gin.Context, params openapi_server.GetFlowsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetFlows",
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

	tmps, err := h.serviceHandler.FlowGets(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a flow list. err: %v", err)
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

func (h *server) GetFlowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetFlowsId",
		"request_address": c.ClientIP,
		"flow_id":         id,
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

	res, err := h.serviceHandler.FlowGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutFlowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "flowsIDPUT",
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

	var req openapi_server.PutFlowsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	actions := []fmaction.Action{}
	for _, v := range req.Actions {
		actions = append(actions, ConvertFlowManagerAction(v))
	}

	f := &fmflow.Flow{
		ID:      target,
		Name:    req.Name,
		Detail:  req.Detail,
		Actions: actions,
	}

	res, err := h.serviceHandler.FlowUpdate(c.Request.Context(), &a, f)
	if err != nil {
		log.Errorf("Could not update the flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteFlowsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteFlowsId",
		"request_address": c.ClientIP,
		"flow_id":         id,
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

	res, err := h.serviceHandler.FlowDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
