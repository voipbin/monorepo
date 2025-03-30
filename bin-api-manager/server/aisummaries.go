package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	amsummary "monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostAisummaries(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAisummaries",
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

	var req openapi_server.PostAisummariesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	onEndFlowID := uuid.FromStringOrNil(req.OnEndFlowId)
	referenceID := uuid.FromStringOrNil(req.ReferenceId)

	res, err := h.serviceHandler.AISummaryCreate(c.Request.Context(), &a, onEndFlowID, amsummary.ReferenceType(req.ReferenceType), referenceID, req.Language)
	if err != nil {
		log.Errorf("Could not create AI. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetAisummaries(c *gin.Context, params openapi_server.GetAisummariesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAisummaries",
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

	tmps, err := h.serviceHandler.AISummaryGetsByCustomerID(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a ai summary list. err: %v", err)
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

func (h *server) GetAisummariesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAisummariesId",
		"request_address": c.ClientIP,
		"ai_id":           id,
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

	res, err := h.serviceHandler.AISummaryGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get an AI. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteAisummariesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteAisummariesId",
		"request_address": c.ClientIP,
		"ai_id":           id,
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

	res, err := h.serviceHandler.AISummaryDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the ai summary. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
