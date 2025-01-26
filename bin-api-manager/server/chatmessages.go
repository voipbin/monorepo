package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	cmmedia "monorepo/bin-chat-manager/models/media"
	cmchatmessage "monorepo/bin-chat-manager/models/messagechat"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostChatmessages(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostChatmessages",
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

	var req openapi_server.PostChatmessagesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	chatID := uuid.FromStringOrNil(req.ChatId)
	source := ConvertCommonAddress(req.Source)

	medias := []cmmedia.Media{}
	if req.Medias != nil {
		for _, m := range *req.Medias {
			medias = append(medias, ConvertChatManagerMedia(m))
		}
	}

	res, err := h.serviceHandler.ChatmessageCreate(c.Request.Context(), &a, chatID, source, cmchatmessage.Type(req.Type), req.Text, medias)
	if err != nil {
		log.Errorf("Could not create a chatmessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetChatmessages(c *gin.Context, params openapi_server.GetChatmessagesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetChatmessages",
		"request_address": c.ClientIP,
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
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	chatID := uuid.FromStringOrNil(params.ChatId)
	if chatID == uuid.Nil {
		log.Errorf("Invalid chat id. chat_id: %s", params.ChatId)
		c.AbortWithStatus(400)
		return
	}

	tmps, err := h.serviceHandler.ChatmessageGetsByChatID(c.Request.Context(), &a, chatID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a chatmessage list. err: %v", err)
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

func (h *server) GetChatmessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetChatmessagesId",
		"request_address": c.ClientIP,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ChatmessageGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a chatmessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteChatmessagesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteChatmessagesId",
		"request_address": c.ClientIP,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.ChatmessageDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete a chatmessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
