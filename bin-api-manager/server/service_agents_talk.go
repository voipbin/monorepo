package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// GetServiceAgentsTalkChats handles GET /service_agents/talk_chats
func (h *server) GetServiceAgentsTalkChats(c *gin.Context, params openapi_server.GetServiceAgentsTalkChatsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsTalkChats",
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

	tmps, err := h.serviceHandler.ServiceAgentTalkList(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get talks info. err: %v", err)
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

// PostServiceAgentsTalkChats handles POST /service_agents/talk_chats
func (h *server) PostServiceAgentsTalkChats(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsTalkChats",
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

	var req openapi_server.PostServiceAgentsTalkChatsJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// Convert type (req.Type is non-pointer, required field)
	talkType := tkchat.Type(req.Type)

	res, err := h.serviceHandler.ServiceAgentTalkCreate(c.Request.Context(), &a, talkType)
	if err != nil {
		log.Errorf("Could not create talk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// GetServiceAgentsTalkChatsId handles GET /service_agents/talk_chats/{id}
func (h *server) GetServiceAgentsTalkChatsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsTalkChatsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get talk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// DeleteServiceAgentsTalkChatsId handles DELETE /service_agents/talk_chats/{id}
func (h *server) DeleteServiceAgentsTalkChatsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsTalkChatsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete talk. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// GetServiceAgentsTalkChatsIdParticipants handles GET /service_agents/talk_chats/{id}/participants
func (h *server) GetServiceAgentsTalkChatsIdParticipants(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsTalkChatsIdParticipants",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkParticipantList(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get participants. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// PostServiceAgentsTalkChatsIdParticipants handles POST /service_agents/talk_chats/{id}/participants
func (h *server) PostServiceAgentsTalkChatsIdParticipants(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsTalkChatsIdParticipants",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostServiceAgentsTalkChatsIdParticipantsJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	ownerID := uuid.FromStringOrNil(req.OwnerId)
	if ownerID == uuid.Nil {
		log.Error("Could not parse owner_id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkParticipantCreate(c.Request.Context(), &a, target, req.OwnerType, ownerID)
	if err != nil {
		log.Errorf("Could not add participant. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// DeleteServiceAgentsTalkChatsIdParticipantsParticipantId handles DELETE /service_agents/talk_chats/{id}/participants/{participant_id}
func (h *server) DeleteServiceAgentsTalkChatsIdParticipantsParticipantId(c *gin.Context, id string, participantId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsTalkChatsIdParticipantsParticipantId",
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

	talkID := uuid.FromStringOrNil(id)
	if talkID == uuid.Nil {
		log.Error("Could not parse talk id.")
		c.AbortWithStatus(400)
		return
	}

	participantID := uuid.FromStringOrNil(participantId)
	if participantID == uuid.Nil {
		log.Error("Could not parse participant id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkParticipantDelete(c.Request.Context(), &a, talkID, participantID)
	if err != nil {
		log.Errorf("Could not delete participant. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// GetServiceAgentsTalkMessages handles GET /service_agents/talk_messages
func (h *server) GetServiceAgentsTalkMessages(c *gin.Context, params openapi_server.GetServiceAgentsTalkMessagesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsTalkMessages",
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

	tmps, err := h.serviceHandler.ServiceAgentTalkMessageList(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get messages info. err: %v", err)
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

// PostServiceAgentsTalkMessages handles POST /service_agents/talk_messages
func (h *server) PostServiceAgentsTalkMessages(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsTalkMessages",
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

	var req openapi_server.PostServiceAgentsTalkMessagesJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	chatID := uuid.FromStringOrNil(req.ChatId)
	if chatID == uuid.Nil {
		log.Error("Could not parse chat_id.")
		c.AbortWithStatus(400)
		return
	}

	// Parse parent_id if provided (ParentId is pointer/optional)
	var parentID *uuid.UUID
	if req.ParentId != nil {
		pid := uuid.FromStringOrNil(*req.ParentId)
		if pid != uuid.Nil {
			parentID = &pid
		}
	}

	// Convert type (req.Type is non-pointer, required field)
	msgType := tkmessage.Type(req.Type)

	res, err := h.serviceHandler.ServiceAgentTalkMessageCreate(c.Request.Context(), &a, chatID, parentID, msgType, req.Text)
	if err != nil {
		log.Errorf("Could not create message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// GetServiceAgentsTalkMessagesId handles GET /service_agents/talk_messages/{id}
func (h *server) GetServiceAgentsTalkMessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsTalkMessagesId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkMessageGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// DeleteServiceAgentsTalkMessagesId handles DELETE /service_agents/talk_messages/{id}
func (h *server) DeleteServiceAgentsTalkMessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsTalkMessagesId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkMessageDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// PostServiceAgentsTalkMessagesIdReactions handles POST /service_agents/talk_messages/{id}/reactions
func (h *server) PostServiceAgentsTalkMessagesIdReactions(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsTalkMessagesIdReactions",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostServiceAgentsTalkMessagesIdReactionsJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("Could not parse request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ServiceAgentTalkMessageReactionCreate(c.Request.Context(), &a, target, req.Emoji)
	if err != nil {
		log.Errorf("Could not add reaction. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
