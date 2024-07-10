package service_agents

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cvmedia "monorepo/bin-conversation-manager/models/media"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// conversationsGET handles GET service_talk/conversations request.
// It returns list of calls of the given agent.

// @Summary		Get list of conversations
// @Description	get conversations of the agent
// @Produce		json
// @Param			page_size	query		int		false	"The size of results. Max 100"
// @Param			page_token	query		string	false	"The token. tm_create"
// @Success		200			{object}	response.BodyServiceAgentConversationsGET
// @Router			/v1.0/service_agents/conversations [get]
func conversationsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationsGET",
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

	var requestParam request.ParamServiceAgentsConversationsGET
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

	// get
	tmps, err := serviceHandler.ServiceAgentConversationGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get calls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyServiceAgentsConversationsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// conversationsIDGET handles GET /service_agents/conversations/{id} request.
// It returns detail call info.
//
//	@Summary		Get detail call info.
//	@Description	Returns detail call info of the given call id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the call"
//	@Success		200	{object}	conversation.Conversation
//	@Router			/v1.0/service_agents/conversations/{id} [get]
func conversationsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationsIDGET",
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
	log = log.WithField("call_id", id)
	log.Debug("Executing callsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentConversationGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conversationsIDMessagesGet handles GET /conversations/{id}/messages request.
// It gets a list of conversation messages with the given info.
//
//	@Summary		Gets a list of conversation messages.
//	@Description	Gets a list of conversation messages
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyConversationsIDMessagesGET
//	@Router			/v1.0/service_agents/conversations/{id}/messages [get]
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

	var req request.ParamServiceAgentsConversationsIDMessagesGET
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
	log.Debug("Executing conversationsIDMessagesGet.")

	// get tmpRes
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	tmpRes, err := serviceHandler.ServiceAgentConversationMessageGets(c.Request.Context(), &a, id, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a conversation message list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmpRes) > 0 {
		nextToken = tmpRes[len(tmpRes)-1].TMCreate
	}
	res := response.BodyServiceAgentsConversationsIDMessagesGET{
		Result: tmpRes,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// conversationsIDMessagesPost handles POST /service_agents/conversations/<conversation-id>/messages request.
// It sends a message with the given info and returns sent message info.
//
//	@Summary		Send a message and returns detail sent message info.
//	@Description	Send a message and returns a sent message info.
//	@Produce		json
//	@Param			message	body		request.BodyConversationsIDMessagesPOST	true	"message info."
//	@Success		200		{object}	message.WebhookMessage
//	@Router			/v1.0/service_agents/conversations/{id}/messages [post]
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

	var req request.BodyServiceAgentsConversationsIDMessagesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing conversationsIDMessagesPost.")

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("target_id", id)
	log.Debug("Executing conversationsIDMessagesPost.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ServiceAgentConversationMessageSend(c.Request.Context(), &a, id, req.Text, []cvmedia.Media{})
	if err != nil {
		log.Errorf("Could not create a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
