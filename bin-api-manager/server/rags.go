package server

import (
	"github.com/gofrs/uuid"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
)

func (h *server) PostRagsQuery(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRagsQuery",
		"request_address": c.ClientIP(),
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

	var req struct {
		RagID uuid.UUID `json:"rag_id"`
		Query string    `json:"query"`
		TopK  *int      `json:"top_k,omitempty"`
	}
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	topK := 0
	if req.TopK != nil {
		topK = *req.TopK
	}

	res, err := h.serviceHandler.RagQuery(c.Request.Context(), &a, req.RagID, req.Query, topK)
	if err != nil {
		log.Errorf("Could not query RAG. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
