package conversationaccounts

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// conversationAccountsGet handles GET /conversation_accounts request.
// It gets a list of conversation accounts with the given info.
//	@Summary		Gets a list of conversation accounts.
//	@Description	Gets a list of conversation accounts
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyConversationsGET
//	@Router			/v1.0/conversation_accounts [get]
func conversationAccountsGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationAccountsGet",
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

	var req request.ParamConversationAccountsGET
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
	tmpRes, err := serviceHandler.ConversationAccountGetsByCustomerID(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a conversation list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmpRes) > 0 {
		nextToken = tmpRes[len(tmpRes)-1].TMCreate
	}
	res := response.BodyConversationAccountsGET{
		Result: tmpRes,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// conversationAccountsPost handles POST /conversation_accounts request.
// It creates a new conversation account with the given info and returns created conversation account info.
//	@Summary		Create a new conversation account and returns detail created conversation account info.
//	@Description	Create a new conversation account and returns detail created conversation account info.
//	@Produce		json
//	@Param			customer	body		request.BodyConversationAccountsPOST	true	"customer info."
//	@Success		200			{object}	account.WebhookMessage
//	@Router			/v1.0/conversation_accounts [post]
func conversationAccountsPost(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationAccountsPost",
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

	var req request.BodyConversationAccountsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Creating a customer.")

	// create a customer
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ConversationAccountCreate(
		c.Request.Context(),
		&a,
		req.Type,
		req.Name,
		req.Detail,
		req.Secret,
		req.Token,
	)
	if err != nil {
		log.Errorf("Could not create a conversation account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conversationAccountsIDGet handles GET /conversation_accounts/{id} request.
// It returns detail conversation account info.
//	@Summary		Returns detail conversation account info.
//	@Description	Returns detail conversation account info of the given conversation account id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the conversation account"
//	@Success		200	{object}	account.WebhookMessage
//	@Router			/v1.0/conversation_accounts/{id} [get]
func conversationAccountsIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationAccountsIDGet",
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
	res, err := serviceHandler.ConversationAccountGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a customer. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conversationAccountsIDPut handles PUT /conversation_accounts/{id} request.
// It returns detail conversation account info.
//	@Summary		Returns detail conversation account info.
//	@Description	Returns detail conversation account info of the given conversation account id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the conversation account"
//	@Success		200	{object}	account.WebhookMessage
//	@Router			/v1.0/conversation_accounts/{id} [put]
func conversationAccountsIDPut(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationAccountsIDGet",
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

	var req request.BodyConversationAccountsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Updating a conversation account.")

	log.Debug("Executing conversationAccountsIDPut.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ConversationAccountUpdate(c.Request.Context(), &a, id, req.Name, req.Detail, req.Secret, req.Token)
	if err != nil {
		log.Errorf("Could not update the conversation account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conversationAccountsIDDelete handles DELETE /conversation_accounts/{id} request.
// It returns detail conversation account info.
//	@Summary		Returns detail conversation account info.
//	@Description	Returns detail conversation account info of the given conversation account id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the conversation account"
//	@Success		200	{object}	account.WebhookMessage
//	@Router			/v1.0/conversation_accounts/{id} [put]
func conversationAccountsIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conversationAccountsIDDelete",
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

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ConversationAccountDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the conversation account. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
