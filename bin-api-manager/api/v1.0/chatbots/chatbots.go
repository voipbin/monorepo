package chatbots

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// chatbotsPOST handles POST /chatbots request.
// It creates a new chatbot with the given info and returns created chatbot info.
//	@Summary		Create a new chatbot and returns detail created chatbot info.
//	@Description	Create a new chatbot and returns detail created chatbot info.
//	@Produce		json
//	@Param			chatbot	body		request.BodyChatbotsPOST	true	"chatbot info."
//	@Success		200		{object}	chatbot.WebhookMessage
//	@Router			/v1.0/chatbots [post]
func chatbotsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatbotsPOST",
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

	var req request.BodyChatbotsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing chatbotsPOST.")

	// create a chatbot
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatbotCreate(c.Request.Context(), &a, req.Name, req.Detail, req.EngineType, req.InitPrompt)
	if err != nil {
		log.Errorf("Could not create a chatbot. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatbotsGET handles GET /chatbots request.
// It gets a list of chatbot with the given info.
//	@Summary		Gets a list of chatbots.
//	@Description	Gets a list of chatbots
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyChatbotsGET
//	@Router			/v1.0/chatbots [get]
func chatbotsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatbotsGET",
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

	var req request.ParamChatbotsGET
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
	log.Debugf("chatbotsGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get chatbots
	chatbots, err := serviceHandler.ChatbotGetsByCustomerID(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a chatbot list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(chatbots) > 0 {
		nextToken = chatbots[len(chatbots)-1].TMCreate
	}
	res := response.BodyChatbotsGET{
		Result: chatbots,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// chatbotsIDGET handles GET /chatbots/{id} request.
// It returns detail chatbot info.
//	@Summary		Returns detail chatbot info.
//	@Description	Returns detail chatbot info of the given chatbot id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatbot"
//	@Success		200	{object}	chatbot.Chatbot
//	@Router			/v1.0/chatbots/{id} [get]
func chatbotsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatbotsIDGET",
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
	log = log.WithField("chatbot_id", id)
	log.Debug("Executing chatbotsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatbotGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a chatbot. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatbotsIDDELETE handles DELETE /chatbots/{id} request.
// It deletes a exist chatbot info.
//	@Summary		Delete a existing chatbot.
//	@Description	Delete a existing chatbot.
//	@Produce		json
//	@Param			id	query	string	true	"The chatbot's id"
//	@Success		200
//	@Router			/v1.0/chatbots/{id} [delete]
func chatbotsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatbotsIDDELETE",
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
	log = log.WithField("chatbot_id", id)
	log.Debug("Executing chatbotsIDDELETE.")

	// delete an chatbot
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatbotDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the chatbot. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatbotsIDPUT handles PUT /chatbots/<chatbot-id> request.
// It updates the existed chatbot with the given info and returns updated chatbot info.
//	@Summary		Update the chatbot and returns updated chatbot info.
//	@Description	Update the chatbot and returns updated chatbot info.
//	@Produce		json
//	@Param			chatbot	body		request.BodyChatbotsIDPUT	true	"chatbot info."
//	@Success		200		{object}	chatbot.WebhookMessage
//	@Router			/v1.0/chatbots/{id} [put]
func chatbotsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatbotsIDPUT",
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

	var req request.BodyChatbotsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("chatbot_id", id)
	log.Debug("Executing chatbotsIDPUT.")

	// update the chatbot
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatbotUpdate(c.Request.Context(), &a, id, req.Name, req.Detail, req.EngineType, req.InitPrompt)
	if err != nil {
		log.Errorf("Could not update the chatbot. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
