package chats

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// chatsPOST handles POST /chats request.
// It creates a new chat with the given info and returns created chat info.
//	@Summary		Create a new chat and returns detail created chat info.
//	@Description	Create a new chat and returns detail created chat info.
//	@Produce		json
//	@Param			chat	body		request.BodyChatsPOST	true	"chat info."
//	@Success		200		{object}	chat.WebhookMessage
//	@Router			/v1.0/chats [post]
func chatsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatsPOST",
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

	var req request.BodyChatsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatsPOST.")

	// create
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatCreate(c.Request.Context(), &a, req.Type, req.OwnerID, req.ParticipantID, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create a chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatsGET handles GET /chats request.
// It gets a list of chats with the given info.
//	@Summary		Gets a list of chats.
//	@Description	Gets a list of chats
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyChatsGET
//	@Router			/v1.0/chats [get]
func chatsGET(c *gin.Context) {
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

	var req request.ParamChatsGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("chatsGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get list
	tmps, err := serviceHandler.ChatGetsByCustomerID(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a chat list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyChatsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// chatsIDGET handles GET /chats/{id} request.
// It returns detail chat info.
//	@Summary		Returns detail chat info.
//	@Description	Returns detail chat info of the given chat id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chat"
//	@Success		200	{object}	chat.Chat
//	@Router			/v1.0/chats/{id} [get]
func chatsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatsIDGET",
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
	log = log.WithField("chat_id", id)
	log.Debug("Executing chatsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatsIDDELETE handles DELETE /chats/{id} request.
// It deletes a exist chat info.
//	@Summary		Delete a existing chat.
//	@Description	Delete a existing chat.
//	@Produce		json
//	@Param			id	query	string	true	"The chat's id"
//	@Success		200
//	@Router			/v1.0/chats/{id} [delete]
func chatsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatsIDDELETE",
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
	log = log.WithField("chat_id", id)
	log.Debug("Executing chatsIDDELETE.")

	// delete
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatsIDPUT handles PUT /chats/{id} request.
// It updates a exist chat info with the given info.
// And returns updated chat info if it succeed.
//	@Summary		Update a chat and reuturns updated info.
//	@Description	Update a chat and returns detail updated info.
//	@Produce		json
//	@Param			id			query		string					true	"The chat's id"
//	@Param			update_info	body		request.BodyChatsIDPUT	true	"The update info"
//	@Success		200			{object}	chat.Chat
//	@Router			/v1.0/chats/{id} [put]
func chatsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatsIDPUT",
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
	log = log.WithField("chat_id", id)

	var req request.BodyChatsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatsIDPUT.")

	// update
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatUpdateBasicInfo(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatsIDOwnerIDPUT handles PUT /chats/{id}/owner_id request.
// It updates a exist chat info with the given chat info.
// And returns updated chat info if it succeed.
//	@Summary		Update a chat and reuturns updated info.
//	@Description	Update a chat and returns detail updated info.
//	@Produce		json
//	@Param			id			query		string							true	"The chat's id"
//	@Param			update_info	body		request.BodyChatsIDOwnerIDPUT	true	"The update info"
//	@Success		200			{object}	chat.Chat
//	@Router			/v1.0/chats/{id}/owner_id [put]
func chatsIDOwnerIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatsIDOwnerIDPUT",
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
	log = log.WithField("chat_id", id)

	var req request.BodyChatsIDOwnerIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatsIDOwnerIDPUT.")

	// update
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatUpdateOwnerID(c.Request.Context(), &a, id, req.OwnerID)
	if err != nil {
		log.Errorf("Could not update the owner_id. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatsIDParticipantIDsPOST handles POST /chats/{id}/participant_ids request.
// It adds a given participant id to the chat and returns updated chat info.
//	@Summary		Add a new participant id to the chat and returns detail updated chat info.
//	@Description	Add a new participant id to the cat and returns detail updated chat info.
//	@Produce		json
//	@Param			chat	body		request.BodyChatsPOST	true	"chat info."
//	@Success		200		{object}	chat.WebhookMessage
//	@Router			/v1.0/chats/{id}/participant_ids [post]
func chatsIDParticipantIDsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatsPOST",
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
	log = log.WithField("chat_id", id)

	var req request.BodyChatsIDParticipantIDsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatsIDParticipantIDsPOST.")

	// create
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatAddParticipantID(c.Request.Context(), &a, id, req.ParticipantID)
	if err != nil {
		log.Errorf("Could not add a participant id. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatsIDParticipantIDsIDDELETE handles DELETE /chats/{id}/participant_ids/{participant_id} request.
// It removes a gieven participant id from the chat.
//	@Summary		Remove participant id.
//	@Description	Remove participant id.
//	@Produce		json
//	@Param			id				query	string	true	"The chat's id"
//	@Param			participant_id	query	string	true	"The chat's participant id"
//	@Success		200
//	@Router			/v1.0/chats/{id}/participant_ids/{participant_id} [delete]
func chatsIDParticipantIDsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatsIDParticipantIDsIDDELETE",
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
	participantID := uuid.FromStringOrNil(c.Params.ByName("participant_id"))
	log = log.WithFields(logrus.Fields{
		"chat_id":        id,
		"participant_id": participantID,
	})

	log.Debug("Executing chatsIDParticipantIDsIDDELETE.")

	// remove
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatRemoveParticipantID(c.Request.Context(), &a, id, participantID)
	if err != nil {
		log.Errorf("Could not remove the participant id. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
