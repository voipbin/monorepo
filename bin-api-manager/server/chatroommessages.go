package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/gens/openapi_server"
	chmedia "monorepo/bin-chat-manager/models/media"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostChatroommessages(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostChatroommessages",
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

	var req openapi_server.PostChatroommessagesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	chatroomID := uuid.FromStringOrNil(req.ChatroomId)
	if chatroomID == uuid.Nil {
		log.Errorf("Invalid chatroom id.")
		c.AbortWithStatus(400)
		return
	}

	medias := []chmedia.Media{}
	if req.Medias != nil {
		for _, v := range *req.Medias {
			medias = append(medias, ConvertChatManagerMedia(v))
		}
	}

	res, err := h.serviceHandler.ChatroommessageCreate(c.Request.Context(), &a, chatroomID, req.Text, medias)
	if err != nil {
		log.Errorf("Could not create a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetChatroommessages(c *gin.Context, params openapi_server.GetChatroommessagesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetChatroommessages",
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

	chatroomID := uuid.FromStringOrNil(params.ChatroomId)

	tmps, err := h.serviceHandler.ChatroommessageGetsByChatroomID(c.Request.Context(), &a, chatroomID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a chatroommessage list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyChatroommessagesGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

func (h *server) GetChatroommessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetChatroommessagesId",
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

	res, err := h.serviceHandler.ChatroommessageGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteChatroommessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroommessagesIDDELETE",
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

	res, err := h.serviceHandler.ChatroommessageDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
