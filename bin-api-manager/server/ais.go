package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostAis(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAis",
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

	var req openapi_server.PostAisJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// TODO: Pass toolNames when OpenAPI schema is updated to include tool_names field
	res, err := h.serviceHandler.AICreate(
		c.Request.Context(),
		&a,
		req.Name,
		req.Detail,
		amai.EngineType(req.EngineType),
		amai.EngineModel(req.EngineModel),
		req.EngineData,
		req.EngineKey,
		req.InitPrompt,
		amai.TTSType(req.TtsType),
		req.TtsVoiceId,
		amai.STTType(req.SttType),
		nil, // toolNames - not yet exposed in OpenAPI
	)
	if err != nil {
		log.Errorf("Could not create a AI. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetAis(c *gin.Context, params openapi_server.GetAisParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAis",
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

	tmps, err := h.serviceHandler.AIGetsByCustomerID(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a AI list. err: %v", err)
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

func (h *server) GetAisId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAisId",
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

	res, err := h.serviceHandler.AIGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get an AI. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteAisId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteAisId",
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

	res, err := h.serviceHandler.AIDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the ai. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAisId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAisId",
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

	var req openapi_server.PutAisIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// TODO: Pass toolNames when OpenAPI schema is updated to include tool_names field
	res, err := h.serviceHandler.AIUpdate(
		c.Request.Context(),
		&a,
		target,
		req.Name,
		req.Detail,
		amai.EngineType(req.EngineType),
		amai.EngineModel(req.EngineModel),
		req.EngineData,
		req.EngineKey,
		req.InitPrompt,
		amai.TTSType(req.TtsType),
		req.TtsVoiceId,
		amai.STTType(req.SttType),
		nil, // toolNames - not yet exposed in OpenAPI
	)
	if err != nil {
		log.Errorf("Could not update the ai. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
