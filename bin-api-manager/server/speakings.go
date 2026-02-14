package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	openapi_server "monorepo/bin-api-manager/gens/openapi_server"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"
)

// GetSpeakings implements GET /v1/speakings
func (h *server) GetSpeakings(c *gin.Context, params openapi_server.GetSpeakingsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetSpeakings",
		"request_address": c.ClientIP(),
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize == 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.SpeakingList(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get speaking list: %v", err)
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

// PostSpeakings implements POST /v1/speakings
func (h *server) PostSpeakings(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostSpeakings",
		"request_address": c.ClientIP(),
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req openapi_server.PostSpeakingsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// Validate reference_type
	referenceType := tmstreaming.ReferenceType(req.ReferenceType)
	switch referenceType {
	case tmstreaming.ReferenceTypeCall, tmstreaming.ReferenceTypeConfbridge:
		// valid
	default:
		log.Errorf("Invalid reference_type: %s", req.ReferenceType)
		c.AbortWithStatus(400)
		return
	}

	referenceID := uuid.FromStringOrNil(req.ReferenceId)
	if referenceID == uuid.Nil {
		log.Errorf("Invalid reference_id")
		c.AbortWithStatus(400)
		return
	}

	language := ""
	if req.Language != nil {
		language = *req.Language
	}
	provider := ""
	if req.Provider != nil {
		provider = *req.Provider
	}
	voiceID := ""
	if req.VoiceId != nil {
		voiceID = *req.VoiceId
	}
	direction := tmstreaming.DirectionNone
	if req.Direction != nil {
		direction = tmstreaming.Direction(*req.Direction)
	}

	// Validate direction
	switch direction {
	case tmstreaming.DirectionNone, tmstreaming.DirectionIncoming, tmstreaming.DirectionOutgoing, tmstreaming.DirectionBoth:
		// valid
	default:
		log.Errorf("Invalid direction: %s", direction)
		c.AbortWithStatus(400)
		return
	}

	speaking, err := h.serviceHandler.SpeakingCreate(c.Request.Context(), &a, referenceType, referenceID, language, provider, voiceID, direction)
	if err != nil {
		log.Errorf("Could not create speaking: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(201, speaking)
}

// DeleteSpeakingsId implements DELETE /v1/speakings/{id}
func (h *server) DeleteSpeakingsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteSpeakingsId",
		"request_address": c.ClientIP(),
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	speakingID := uuid.FromStringOrNil(id)
	if speakingID == uuid.Nil {
		log.Errorf("Invalid speaking id")
		c.AbortWithStatus(400)
		return
	}

	speaking, err := h.serviceHandler.SpeakingDelete(c.Request.Context(), &a, speakingID)
	if err != nil {
		log.Errorf("Could not delete speaking: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, speaking)
}

// GetSpeakingsId implements GET /v1/speakings/{id}
func (h *server) GetSpeakingsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetSpeakingsId",
		"request_address": c.ClientIP(),
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	speakingID := uuid.FromStringOrNil(id)
	if speakingID == uuid.Nil {
		log.Errorf("Invalid speaking id")
		c.AbortWithStatus(400)
		return
	}

	speaking, err := h.serviceHandler.SpeakingGet(c.Request.Context(), &a, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, speaking)
}

// PostSpeakingsIdFlush implements POST /v1/speakings/{id}/flush
func (h *server) PostSpeakingsIdFlush(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostSpeakingsIdFlush",
		"request_address": c.ClientIP(),
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	speakingID := uuid.FromStringOrNil(id)
	if speakingID == uuid.Nil {
		log.Errorf("Invalid speaking id")
		c.AbortWithStatus(400)
		return
	}

	speaking, err := h.serviceHandler.SpeakingFlush(c.Request.Context(), &a, speakingID)
	if err != nil {
		log.Errorf("Could not flush speaking: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, speaking)
}

// PostSpeakingsIdSay implements POST /v1/speakings/{id}/say
func (h *server) PostSpeakingsIdSay(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostSpeakingsIdSay",
		"request_address": c.ClientIP(),
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	speakingID := uuid.FromStringOrNil(id)
	if speakingID == uuid.Nil {
		log.Errorf("Invalid speaking id")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostSpeakingsIdSayJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	if req.Text == "" {
		log.Errorf("Text is empty")
		c.AbortWithStatus(400)
		return
	}

	if len(req.Text) > 5000 {
		log.Errorf("Text too long: %d characters (max 5000)", len(req.Text))
		c.AbortWithStatus(400)
		return
	}

	speaking, err := h.serviceHandler.SpeakingSay(c.Request.Context(), &a, speakingID, req.Text)
	if err != nil {
		log.Errorf("Could not say text: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, speaking)
}

// PostSpeakingsIdStop implements POST /v1/speakings/{id}/stop
func (h *server) PostSpeakingsIdStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostSpeakingsIdStop",
		"request_address": c.ClientIP(),
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	speakingID := uuid.FromStringOrNil(id)
	if speakingID == uuid.Nil {
		log.Errorf("Invalid speaking id")
		c.AbortWithStatus(400)
		return
	}

	speaking, err := h.serviceHandler.SpeakingStop(c.Request.Context(), &a, speakingID)
	if err != nil {
		log.Errorf("Could not stop speaking: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, speaking)
}
