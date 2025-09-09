package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) PostMcp(c *gin.Context) {
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

	var req openapi_server.PostMcpJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log.Debugf("req: %+v", req)

	// actions := []fmaction.Action{}
	// for _, v := range req.Actions {
	// 	actions = append(actions, ConvertFlowManagerAction(v))
	// }

	// res, err := h.serviceHandler.FlowCreate(c.Request.Context(), &a, req.Name, req.Detail, actions, true)
	// if err != nil {
	// 	log.Errorf("Could not create data. err: %v", err)
	// 	c.AbortWithStatus(400)
	// 	return
	// }

	c.JSON(200, nil)
}
