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

// callsGET handles GET service_talk/calls request.
// It returns list of calls of the given customer.

// @Summary		Get list of calls
// @Description	get calls of the customer
// @Produce		json
// @Param			page_size	query		int		false	"The size of results. Max 100"
// @Param			page_token	query		string	false	"The token. tm_create"
// @Success		200			{object}	response.BodyCallsGET
// @Router			/v1.0/service_agents/calls [get]
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
	calls, err := serviceHandler.ServiceAgentCallGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
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

// callsIDGET handles GET /service_agents/calls/{id} request.
// It returns detail call info.
//
//	@Summary		Get detail call info.
//	@Description	Returns detail call info of the given call id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the call"
//	@Success		200	{object}	call.Call
//	@Router			/v1.0/service_agents/calls/{id} [get]
func callsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "callsIDGET",
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
	res, err := serviceHandler.ServiceAgentCallGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
