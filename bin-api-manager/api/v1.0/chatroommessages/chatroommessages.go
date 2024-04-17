package chatroommessages

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// chatroommessagesPOST handles POST /chatroommessages request.
// It creates a new chatroommessage with the given info and returns created chatroommessage info.
//	@Summary		Create a new chatroommessage and returns detail created chatroommessage info.
//	@Description	Create a new chatroommessage and returns detail created chatroommessage info.
//	@Produce		json
//	@Param			chatroommessage	body		request.BodyChatroommessagesPOST	true	"chatroommessage info."
//	@Success		200				{object}	messagechat.WebhookMessage
//	@Router			/v1.0/chatroommessages [post]
func chatroommessagesPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroommessagesPOST",
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

	var req request.BodyChatroommessagesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatroommessagesPOST.")

	// create
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatroommessageCreate(c.Request.Context(), &a, req.ChatroomID, req.Text)
	if err != nil {
		log.Errorf("Could not create a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatroommessagesGET handles GET /chatroommessages request.
// It gets a list of chatroommessages with the given info.
//	@Summary		Gets a list of chatroommessages.
//	@Description	Gets a list of chatroommessages
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Param			chatroom_id	query		string	true	"The id of the chatroom"
//	@Success		200			{object}	response.BodyChatsGET
//	@Router			/v1.0/chatroommessages [get]
func chatroommessagesGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroommessagesGET",
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

	var req request.ParamChatroommessagesGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	chatroomID := uuid.FromStringOrNil(req.ChatroomID)

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("chatroommessagesGET. Received request detail. chatroom_id: %s, page_size: %d, page_token: %s", chatroomID, pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get list
	tmps, err := serviceHandler.ChatroommessageGetsByChatroomID(c.Request.Context(), &a, chatroomID, pageSize, req.PageToken)
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

// chatroommessagesIDGET handles GET /chatroommessages/{id} request.
// It returns detail chatroommessage info.
//	@Summary		Returns detail chatroommessage info.
//	@Description	Returns detail chatroommessage info of the given chatroommessage id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatroommessage"
//	@Success		200	{object}	messagechatroom.Messagechatroom
//	@Router			/v1.0/chatroommessages/{id} [get]
func chatroommessagesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroommessagesIDGET",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("chatroommessage_id", id)
	log.Debug("Executing chatroomsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatroommessageGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatroommessagesIDDELETE handles DELETE /chatroommessages/{id} request.
// It deletes the chatroommessage and returns deleted chatroommessage info.
//	@Summary		Deletes a chatroommessage and returns detail chatroommessage info.
//	@Description	Deletes a chatroommessage and returns detail chatroommessage info.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatroommessage"
//	@Success		200	{object}	messagechatroom.Messagechatroom
//	@Router			/v1.0/chatroommessages/{id} [delete]
func chatroommessagesIDDELETE(c *gin.Context) {
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("chatroommessage_id", id)
	log.Debug("Executing chatroomsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatroommessageDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete a chatroommessage. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
