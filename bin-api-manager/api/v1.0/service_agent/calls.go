package service_talk

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// callsGET handles GET service_talk/calls request.
// It returns list of calls of the given customer.

// @Summary		Get list of calls
// @Description	get calls of the customer
// @Produce		json
// @Param			page_size	query		int		false	"The size of results. Max 100"
// @Param			page_token	query		string	false	"The token. tm_create"
// @Success		200			{object}	response.BodyCallsGET
// @Router			/v1.0/calls [get]
func callsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsGET",
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

	var requestParam request.ParamServiceAgentCallsGET
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

	// get calls
	calls, err := serviceHandler.CallGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get calls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(calls) > 0 {
		nextToken = calls[len(calls)-1].TMCreate
	}
	res := response.BodyServiceAgentCallsGET{
		Result: calls,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// callsPOST handles POST /calls request.
// It creates a temp flow and create a call with temp flow.
//
//	@Summary		Make an outbound call
//	@Description	dialing to destination
//	@Produce		json
//	@Param			call	body		request.BodyCallsPOST	true	"The call detail"
//	@Success		200		{object}	call.Call
//	@Router			/v1.0/calls [post]
func callsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsPOST",
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

	var req request.BodyServiceAgentCallsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create call
	tmpCalls, tmpGroupcalls, err := serviceHandler.CallCreate(c.Request.Context(), &a, req.FlowID, req.Actions, &req.Source, req.Destinations)
	if err != nil {
		log.Errorf("Could not create a call for outgoing. err; %v", err)
		c.AbortWithStatus(400)
		return
	}

	res := &response.BodyServiceAgentCallsPOST{
		Calls:      tmpCalls,
		Groupcalls: tmpGroupcalls,
	}

	c.JSON(200, res)
}
