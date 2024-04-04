package chatbotcalls

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// chatbotcallsGET handles GET /chatbotcalls request.
// It gets a list of chatbotcalls with the given info.
//	@Summary		Gets a list of chatbotcalls.
//	@Description	Gets a list of chatbotcalls
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyChatbotcallsGET
//	@Router			/v1.0/chatbotcalls [get]
func chatbotcallsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatbotcallsGET",
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

	var req request.ParamChatbotcallsGET
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

	// get chatbotcalls
	chatbots, err := serviceHandler.ChatbotcallGetsByCustomerID(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a chatbot list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(chatbots) > 0 {
		nextToken = chatbots[len(chatbots)-1].TMCreate
	}
	res := response.BodyChatbotcallsGET{
		Result: chatbots,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// chatbotcallsIDGET handles GET /chatbotcalls/{id} request.
// It returns detail chatbotcall info.
//	@Summary		Returns detail chatbotcall info.
//	@Description	Returns detail chatbotcall info of the given chatbotcall id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the chatbotcall"
//	@Success		200	{object}	chatbotcall.Chatbotcall
//	@Router			/v1.0/chatbotcalls/{id} [get]
func chatbotcallsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatbotcallsIDGET",
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
	res, err := serviceHandler.ChatbotcallGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a chatbot. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// chatbotcallsIDDELETE handles DELETE /chatbotcalls/{id} request.
// It deletes a exist chatbot info.
//	@Summary		Delete a existing chatbotcall.
//	@Description	Delete a existing chatbotcall.
//	@Produce		json
//	@Param			id	query	string	true	"The chatbotcall's id"
//	@Success		200
//	@Router			/v1.0/chatbotcalls/{id} [delete]
func chatbotcallsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "chatbotcallsIDDELETE",
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
	log.Debug("Executing chatbotcallsIDDELETE.")

	// delete an chatbot
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.ChatbotcallDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the chatbotcall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
