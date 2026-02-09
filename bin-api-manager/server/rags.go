package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) PostRagsQuery(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRagsQuery",
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

	var req openapi_server.PostRagsQueryJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// Convert doc types
	var docTypes []string
	if req.DocTypes != nil {
		docTypes = make([]string, len(*req.DocTypes))
		for i, dt := range *req.DocTypes {
			docTypes[i] = string(dt)
		}
	}

	// Convert top_k
	topK := 0
	if req.TopK != nil {
		topK = *req.TopK
	}

	res, err := h.serviceHandler.RagQuery(c.Request.Context(), &a, req.Query, docTypes, topK)
	if err != nil {
		log.Errorf("Could not query RAG. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
