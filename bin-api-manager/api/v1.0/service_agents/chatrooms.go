package service_agents

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// chatroomsGET handles GET service_agents/chatrooms request.
// It returns list of chatrooms of the given customer.

// @Summary		Get list of chatrooms
// @Description	get chatrooms of the customer
// @Produce		json
// @Param			page_size	query		int		false	"The size of results. Max 100"
// @Param			page_token	query		string	false	"The token. tm_create"
// @Success		200			{object}	response.BodyCallsGET
// @Router			/v1.0/service_agents/chatrooms [get]
func chatroomsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatroomsGET",
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

	var requestParam request.ParamServiceAgentChatroomsGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("callsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get tmps
	tmps, err := serviceHandler.ServiceAgentChatroomGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get calls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyServiceAgentsChatroomsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// chatroomsPOST handles POST /service_agents/chatrooms request.
// It creates a new chatroom with the given info and returns created chatroom info.
//
//	@Summary		Create a new chatroom and returns detail created chatroom info.
//	@Description	Create a new chatroom and returns detail created chatroom info.
//	@Produce		json
//	@Param			chatroom	body		request.BodyServiceAgentChatroomsPOST	true	"chatroom info."
//	@Success		200			{object}	chatroom.WebhookMessage
//	@Router			/v1.0/service_agents/chatrooms [post]
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

	var req request.BodyServiceAgentChatroomsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatroomsPOST.")

	// create
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentChatroomCreate(c.Request.Context(), &a, req.ParticipantID, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create a chatroom. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatroomsIDGET handles GET /service_agents/chatrooms/{id} request.
// It returns detail chatroom info.
//
//	@Summary		Returns detail chatroom info.
//	@Description	Returns detail chatroom info of the given chatroom id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatroom"
//	@Success		200	{object}	chatroom.Chatroom
//	@Router			/v1.0/service_agents/chatrooms/{id} [get]
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
	res, err := serviceHandler.ServiceAgentChatroomGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatroomsIDDELETE handles DELETE /service_agents/chatrooms/{id} request.
// It deletes the chatroom and returns deleted chatroom info.
//
//	@Summary		Deletes a chatroom and returns detail chatroom info.
//	@Description	Deletes a chatroom and returns detail chatroom info.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatroom"
//	@Success		200	{object}	chatroom.Chatroom
//	@Router			/v1.0/service_agents/chatrooms/{id} [delete]
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
	res, err := serviceHandler.ServiceAgentChatroomDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete a chatroom. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatroomsIDPUT handles PUT /service_agents/chatrooms/{id} request.
// It updates a exist chat info with the given info.
// And returns updated chat info if it succeed.
//
//	@Summary		Update a chat and reuturns updated info.
//	@Description	Update a chat and returns detail updated info.
//	@Produce		json
//	@Param			id			query		string									true	"The chatroom's id"
//	@Param			update_info	body		request.BodyServiceAgentChatroomsIDPUT	true	"The update info"
//	@Success		200			{object}	chat.Chat
//	@Router			/v1.0/service_agents/chatrooms/{id} [put]
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

	var req request.BodyServiceAgentChatroomsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatsIDPUT.")

	// update
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentChatroomUpdateBasicInfo(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the chat. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
