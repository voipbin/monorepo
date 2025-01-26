package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	chmedia "monorepo/bin-chat-manager/models/media"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostServiceAgentsChatroommessages(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func (h *server)": "PostServiceAgentsChatroommessages",
		"request_address":  c.ClientIP,
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

	var req openapi_server.PostServiceAgentsChatroommessagesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	chatroomID := uuid.FromStringOrNil(req.ChatroomId)
	medias := []chmedia.Media{}
	for _, v := range req.Medias {
		medias = append(medias, ConvertChatManagerMedia(v))
	}

	res, err := h.serviceHandler.ServiceAgentChatroommessageCreate(c.Request.Context(), &a, chatroomID, req.Text, medias)
	if err != nil {
		log.Errorf("Could not create a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetServiceAgentsChatroommessages(c *gin.Context, params openapi_server.GetServiceAgentsChatroommessagesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func (h *server)": "GetServiceAgentsChatroommessages",
		"request_address":  c.ClientIP,
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

	chatroomID := uuid.FromStringOrNil(params.ChatroomId)

	tmps, err := h.serviceHandler.ServiceAgentChatroommessageGets(c.Request.Context(), &a, chatroomID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a chatroommessage list. err: %v", err)
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

func (h *server) GetServiceAgentsChatroommessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func (h *server)":   "GetServiceAgentsChatroommessagesId",
		"request_address":    c.ClientIP,
		"chatroommessage_id": id,
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

	res, err := h.serviceHandler.ServiceAgentChatroommessageGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteServiceAgentsChatroommessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func (h *server)":   "DeleteServiceAgentsChatroommessagesId",
		"request_address":    c.ClientIP,
		"chatroommessage_id": id,
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

	res, err := h.serviceHandler.ServiceAgentChatroommessageDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
