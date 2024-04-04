package chatrooms

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// chatroomsPOST handles POST /chatrooms request.
// It creates a new chatroom with the given info and returns created chatroom info.
//	@Summary		Create a new chatroom and returns detail created chatroom info.
//	@Description	Create a new chatroom and returns detail created chatroom info.
//	@Produce		json
//	@Param			chatroom	body		request.BodyChatroomsPOST	true	"chatroom info."
//	@Success		200			{object}	chatroom.WebhookMessage
//	@Router			/v1.0/chatrooms [post]
func chatroomsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroomsPOST",
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

	var req request.BodyChatroomsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatroomsPOST.")

	// create
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatroomCreate(c.Request.Context(), &a, req.ParticipantID, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create a chatroom. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatroomsGET handles GET /chatrooms request.
// It gets a list of chatrooms with the given info.
//	@Summary		Gets a list of chatrooms.
//	@Description	Gets a list of chatrooms
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Param			owner_id	query		string	true	"The id of the chatroom owner"
//	@Success		200			{object}	response.BodyChatsGET
//	@Router			/v1.0/chatrooms [get]
func chatroomsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatsGET",
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

	var req request.ParamChatroomsGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	ownerID := uuid.FromStringOrNil(req.OwnerID)
	if ownerID == uuid.Nil {
		// has no owner id info. use default owner id
		ownerID = a.ID
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("chatroomsGET. Received request detail. owner_id: %s, page_size: %d, page_token: %s", ownerID, pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get list
	tmps, err := serviceHandler.ChatroomGetsByOwnerID(c.Request.Context(), &a, ownerID, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a chatroom list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyChatroomsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// chatroomsIDGET handles GET /chatrooms/{id} request.
// It returns detail chatroom info.
//	@Summary		Returns detail chatroom info.
//	@Description	Returns detail chatroom info of the given chatroom id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatroom"
//	@Success		200	{object}	chatroom.Chatroom
//	@Router			/v1.0/chatrooms/{id} [get]
func chatroomsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroomsIDGET",
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
	log = log.WithField("chatroom_id", id)
	log.Debug("Executing chatroomsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatroomGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatroomsIDDELETE handles DELETE /chatrooms/{id} request.
// It deletes the chatroom and returns deleted chatroom info.
//	@Summary		Deletes a chatroom and returns detail chatroom info.
//	@Description	Deletes a chatroom and returns detail chatroom info.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatroom"
//	@Success		200	{object}	chatroom.Chatroom
//	@Router			/v1.0/chatrooms/{id} [delete]
func chatroomsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroomsIDGET",
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
	log = log.WithField("chatroom_id", id)
	log.Debug("Executing chatroomsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatroomDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete a chatroom. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatroomsIDPUT handles PUT /chatrooms/{id} request.
// It updates a exist chat info with the given info.
// And returns updated chat info if it succeed.
//	@Summary		Update a chat and reuturns updated info.
//	@Description	Update a chat and returns detail updated info.
//	@Produce		json
//	@Param			id			query		string						true	"The chatroom's id"
//	@Param			update_info	body		request.BodyChatroomsIDPUT	true	"The update info"
//	@Success		200			{object}	chat.Chat
//	@Router			/v1.0/chatrooms/{id} [put]
func chatroomsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroomsIDPUT",
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
	log = log.WithField("chatroom_id", id)

	var req request.BodyChatroomsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatsIDPUT.")

	// update
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatroomUpdateBasicInfo(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
