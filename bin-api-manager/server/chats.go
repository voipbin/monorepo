package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	chchat "monorepo/bin-chat-manager/models/chat"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostChats(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostChats",
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

	var req openapi_server.PostChatsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	ownerID := uuid.FromStringOrNil(req.OwnerId)
	if ownerID == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	participantIDs := []uuid.UUID{}
	for _, v := range req.ParticipantIds {
		participantIDs = append(participantIDs, uuid.FromStringOrNil(v))
	}

	res, err := h.serviceHandler.ChatCreate(c.Request.Context(), &a, chchat.Type(req.Type), ownerID, participantIDs, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create a chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetChats(c *gin.Context, params openapi_server.GetChatsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetChats",
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

	tmps, err := h.serviceHandler.ChatGetsByCustomerID(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a chat list. err: %v", err)
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

func (h *server) GetChatsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetChatsId",
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

	res, err := h.serviceHandler.ChatGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteChatsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteChatsId",
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

	res, err := h.serviceHandler.ChatDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutChatsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutChatsId",
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

	var req openapi_server.PutChatsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ChatUpdateBasicInfo(c.Request.Context(), &a, target, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutChatsIdRoomOwnerId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutChatsIdRoomOwnerId",
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

	var req openapi_server.PutChatsIdRoomOwnerIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	ownerID := uuid.FromStringOrNil(req.RoomOwnerId)
	if ownerID == uuid.Nil {
		log.Error("Could not parse the owner id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ChatUpdateRoomOwnerID(c.Request.Context(), &a, target, ownerID)
	if err != nil {
		log.Errorf("Could not update the room_owner_id. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostChatsIdParticipantIds(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostChatsIdParticipantIds",
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

	var req openapi_server.PostChatsIdParticipantIdsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	participantID := uuid.FromStringOrNil(req.ParticipantId)
	if participantID == uuid.Nil {
		log.Error("Could not parse the participant id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ChatAddParticipantID(c.Request.Context(), &a, target, participantID)
	if err != nil {
		log.Errorf("Could not add a participant id. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteChatsIdParticipantIdsParticipantId(c *gin.Context, id string, participantId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteChatsIdParticipantIdsParticipantId",
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

	participantID := uuid.FromStringOrNil(participantId)
	if participantID == uuid.Nil {
		log.Error("Could not parse the participant id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ChatRemoveParticipantID(c.Request.Context(), &a, target, participantID)
	if err != nil {
		log.Errorf("Could not remove the participant id. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
