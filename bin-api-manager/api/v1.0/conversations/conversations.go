package conversations

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	_ "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// conversationsGet handles GET /conversations request.
// It gets a list of conversations with the given info.
//	@Summary		Gets a list of conversations.
//	@Description	Gets a list of conversations
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyConversationsGET
//	@Router			/v1.0/conversations [get]
func conversationsGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationsGet",
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

	var req request.ParamConversationsGET
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
	log.Debugf("Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get tmpRes
	tmpRes, err := serviceHandler.ConversationGetsByCustomerID(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a conversation list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmpRes) > 0 {
		nextToken = tmpRes[len(tmpRes)-1].TMCreate
	}
	res := response.BodyConversationsGET{
		Result: tmpRes,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// conversationsIDGet handles GET /conversations/{id} request.
// It returns detail conversation info.
//	@Summary		Returns detail conversation info.
//	@Description	Returns detail conversation info of the given conversation id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the conversation"
//	@Success		200	{object}	conversation.WebhookMessage
//	@Router			/v1.0/conversations/{id} [get]
func conversationsIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationsIDGet",
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
	log = log.WithField("target_id", id)
	log.Debug("Executing customersIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ConversationGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conversationsIDPut handles PUT /conversations/{id} request.
// It updates the  conversation info.
//	@Summary		Update the conversation info.
//	@Description	Update the conversation info of the given conversation id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the conversation"
//	@Success		200	{object}	conversation.WebhookMessage
//	@Router			/v1.0/conversations/{id} [get]
func conversationsIDPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationsIDPut",
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

	var req request.BodyConversationsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ConversationUpdate(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the conversation. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conversationsIDMessagesGet handles GET /conversations/{id}/messages request.
// It gets a list of conversation messages with the given info.
//	@Summary		Gets a list of conversation messages.
//	@Description	Gets a list of conversation messages
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyConversationsIDMessagesGET
//	@Router			/v1.0/conversations/{id}/messages [get]
func conversationsIDMessagesGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationsIDMessagesGet",
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

	var req request.ParamConversationsIDMessagesGET
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
	log.Debugf("Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)
	log.Debug("Executing customersIDGET.")

	// get tmpRes
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	tmpRes, err := serviceHandler.ConversationMessageGetsByConversationID(c.Request.Context(), &a, id, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a conversation message list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmpRes) > 0 {
		nextToken = tmpRes[len(tmpRes)-1].TMCreate
	}
	res := response.BodyConversationsIDMessagesGET{
		Result: tmpRes,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// conversationsIDMessagesPost handles POST /conversations/<conversation-id>/messages request.
// It sends a message with the given info and returns sent message info.
//	@Summary		Send a message and returns detail sent message info.
//	@Description	Send a message and returns a sent message info.
//	@Produce		json
//	@Param			message	body		request.BodyConversationsIDMessagesPOST	true	"message info."
//	@Success		200		{object}	message.WebhookMessage
//	@Router			/v1.0/conversations/{id}/messages [post]
func conversationsIDMessagesPost(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationsIDMessagesPost",
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

	var req request.BodyConversationsIDMessagesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing conversationsIDMessagesPost.")

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)
	log.Debug("Executing customersIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ConversationMessageSend(c.Request.Context(), &a, id, req.Text, []cvmedia.Media{})
	if err != nil {
		log.Errorf("Could not create a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
