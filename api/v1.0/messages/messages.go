package messages

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/message-manager.git/models/message" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// messagesGET handles GET /messages request.
// It returns list of messages of the given customer.
//	@Summary		List order messages
//	@Description	get messages
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyMessagesGET
//	@Router			/v1.0/messages [get]
func messagesGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "messagesGET",
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

	var req request.ParamMessagesGET
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

	// get messages
	messages, err := serviceHandler.MessageGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get messages list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(messages) > 0 {
		nextToken = messages[len(messages)-1].TMCreate
	}
	res := response.BodyMessagesGET{
		Result: messages,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// messagesPOST handles POST /messages request.
// It sends a message with the given info and returns message.
//	@Summary		Send a message and returns a sent message.
//	@Description	Send a message and returns a sent message.
//	@Produce		json
//	@Param			message	body		request.BodyMessagesPOST	true	"Sending message info."
//	@Success		200		{object}	message.Message
//	@Router			/v1.0/messages [post]
func messagesPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "messagesPOST",
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

	var req request.BodyMessagesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// create a message
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.MessageSend(c.Request.Context(), &a, req.Source, req.Destinations, req.Text)
	if err != nil {
		log.Errorf("Could not send the message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// messagesIDDELETE handles DELETE /messages/<id> request.
// It deletes the given id of message and returns the deleted message.
//	@Summary		Delete message
//	@Description	delete message of the given id and returns deleted item.
//	@Produce		json
//	@Param			id	path		string	true	"The message's id"
//	@Success		200	{object}	message.Message
//	@Router			/v1.0/messages/{id} [delete]
func messagesIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "messagesIDDELETE",
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
	log = log.WithField("message_id", id)
	log.Debugf("Executing messagesIDDELETE.")

	// delete message
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.MessageDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete a message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// messagesIDGET handles GET /messages/<id> request.
// It returns message of the given id.
//	@Summary		Get message
//	@Description	get message of the given id
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the message"
//	@Success		200	{object}	message.Message
//	@Router			/v1.0/messages/{id} [get]
func messagesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "messagesIDGET",
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
	log = log.WithField("message_id", id)
	log.Debugf("Executing messagesIDGET.")

	// get message
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.MessageGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get an message. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
