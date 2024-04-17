package conferencecalls

import (
	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// conferencecallsGET handles GET /conferencecalls request.
// It returns list of conferencecalls of the given customer.
//
//	@Summary		Get list of conferencecalls
//	@Description	get conferences of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyCallsGET
//	@Router			/v1.0/conferencecalls [get]
func conferencecallsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencecallsGET",
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

	var requestParam request.ParamConferencecallsGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("conferencecallsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get conferences
	confs, err := serviceHandler.ConferencecallGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not create a flow for outoing call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(confs) > 0 {
		nextToken = confs[len(confs)-1].TMCreate
	}

	res := response.BodyConferencecallsGET{
		Result: confs,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// conferencecallsIDGET handles GET /conferencecalls/{id} request.
// It returns detail conferencecall info.
//
//	@Summary		Returns detail conferencecall info.
//	@Description	Returns detail conferencecall info of the given conferencecall id.
//	@Produce		json
//	@Param			id		path		string	true	"The ID of the conferencecall"
//	@Param			token	query		string	true	"JWT token"
//	@Success		200		{object}	conferencecall.Conferencecall
//	@Router			/v1.0/conferencecalls/{id} [get]
func conferencecallsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencecallsIDGET",
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
	log = log.WithField("conferencecall_id", id)
	log.Debug("Executing conferencecallsIDGET.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferencecallGet(c.Request.Context(), &a, id)
	if err != nil || res == nil {
		log.Errorf("Could not get the conferencecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// conferencecallsIDDELETE handles DELETE /conferencecalls/{id} request.
// It kicks the conferencecall from the conference.
//
//	@Summary		Kicks the conferencecall from the conference.
//	@Description	Kicks the conferencecall.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the conferencecall"
//	@Success		200
//	@Router			/v1.0/conferencecalls/{id} [delete]
func conferencecallsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencecallsIDDELETE",
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
	log = log.WithField("conferencecall_id", id)
	log.Debug("Executing conferencecallsIDDELETE.")

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.ConferencecallKick(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not kick the conferencecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	c.JSON(200, res)
}
