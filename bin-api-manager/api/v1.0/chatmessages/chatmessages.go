package chatmessages

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// chatmessagesPOST handles POST /chatmessages request.
// It creates a new chatmessage with the given info and returns created chatmessage info.
//	@Summary		Create a new chatmessage and returns detail created chatmessage info.
//	@Description	Create a new chatmessage and returns detail created chatmessage info.
//	@Produce		json
//	@Param			chatmessage	body		request.BodyChatmessagesPOST	true	"chatmessage info."
//	@Success		200			{object}	messagechat.WebhookMessage
//	@Router			/v1.0/chatmessages [post]
func chatmessagesPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatmessagesPOST",
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

	var req request.BodyChatmessagesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatmessagesPOST.")

	// create
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatmessageCreate(c.Request.Context(), &a, req.ChatID, req.Source, req.Type, req.Text, req.Medias)
	if err != nil {
		log.Errorf("Could not create a chatmessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatmessagesGET handles GET /chatmessages request.
// It gets a list of chatmessages with the given info.
//	@Summary		Gets a list of chatmessages.
//	@Description	Gets a list of chatmessages
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyChatmessagesGET
//	@Router			/v1.0/chatmessages [get]
func chatmessagesGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatmessagesGET",
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

	var req request.ParamChatmessagesGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	chatID := uuid.FromStringOrNil(req.ChatID)

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("chatmessagesGET. Received request detail. chat_id: %s, page_size: %d, page_token: %s", chatID, pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get list
	tmps, err := serviceHandler.ChatmessageGetsByChatID(c.Request.Context(), &a, chatID, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a chatmessage list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyChatmessagesGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// chatmessagesIDGET handles GET /chatmessages/{id} request.
// It returns detail chatmessage info.
//	@Summary		Returns detail chatmessage info.
//	@Description	Returns detail chatmessage info of the given chatmessage id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatmessage"
//	@Success		200	{object}	messagechat.Messagechat
//	@Router			/v1.0/chatmessages/{id} [get]
func chatmessagesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatmessagesIDGET",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("chatmessage_id", id)
	log.Debug("Executing chatmessagesIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatmessageGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a chatmessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatmessagesIDDELETE handles DELETE /chatmessages/{id} request.
// It deletes the chatmessage and returns deleted chatmessage info.
//	@Summary		Deletes a chatmessage and returns detail chatmessage info.
//	@Description	Deletes a chatmessage and returns detail chatmessage info.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatroom"
//	@Success		200	{object}	messagechat.Messagechat
//	@Router			/v1.0/chatmessages/{id} [delete]
func chatmessagesIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatmessagesIDDELETE",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("chatmessage_id", id)
	log.Debug("Executing chatmessagesIDDELETE.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatmessageDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete a chatmessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
