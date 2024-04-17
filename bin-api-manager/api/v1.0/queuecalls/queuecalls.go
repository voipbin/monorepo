package queuecalls

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

// queuecallsGET handles GET /queuecalls request.
// It returns list of queuecalls of the given customer.
//
//	@Summary		List qeueucalls
//	@Description	get queuecalls of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyQueuecallsGET
//	@Router			/v1.0/queues [get]
func queuecallsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsGET",
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

	var req request.ParamQueuecallsGET
	if err := c.BindQuery(&req); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	} else if pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to max. page_size: %d", pageSize)
	}

	// get tmps
	tmps, err := serviceHandler.QueuecallGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		logrus.Errorf("Could not get queuecalls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyQueuecallsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// queuecallsIDGET handles GET /queuecalls/{id} request.
// It returns detail queuecall info.
//
//	@Summary		Returns detail queuecall info.
//	@Description	Returns detail conferencecall info of the given queuecall id.
//	@Produce		json
//	@Param			id		path		string	true	"The ID of the queuecall"
//	@Param			token	query		string	true	"JWT token"
//	@Success		200		{object}	queuecall.Queuecall
//	@Router			/v1.0/queuecall/{id} [get]
func queuecallsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsIDGET",
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
	log = log.WithField("queuecall_id", id)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.QueuecallGet(c.Request.Context(), &a, id)
	if err != nil || res == nil {
		log.Errorf("Could not get the conferencecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// queuecallsIDDELETE handles DELETE /queuecalls/{id} request.
// It deletes the queuecall.
//
//	@Summary		Deletes the queuecall.
//	@Description	Deletes the queuecall.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the queuecall"
//	@Success		200
//	@Router			/v1.0/queuecalls/{id} [delete]
func queuecallsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsIDDELETE",
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
	log = log.WithField("queuecall_id", id)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.QueuecallDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	c.JSON(200, res)
}

// queuecallsIDKickPOST handles POST /queuecalls/{id}/kick request.
// It kicks the queuecall from the queue.
//
//	@Summary		Kicks the queuecall from the queue.
//	@Description	Kicks the queuecall.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the queuecall"
//	@Success		200
//	@Router			/v1.0/queuecalls/{id}/kick [post]
func queuecallsIDKickPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsIDKickPOST",
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
	log = log.WithField("queuecall_id", id)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.QueuecallKick(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	c.JSON(200, res)
}

// queuecallsReferenceIDIDKickPOST handles POST /queuecalls/reference_id/{id}/kick request.
// It kicks the queuecall of the given reference id from the queue.
//
//	@Summary		Kicks the queuecall of the given reference id from the queue.
//	@Description	Kicks the queuecall of the given reference id.
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the queuecall"
//	@Success		200
//	@Router			/v1.0/queuecalls/reference_id/{id}/kick [post]
func queuecallsReferenceIDIDKickPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuecallsReferenceIDIDKickPOST",
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

	// get referenceID
	referenceID := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("queuecall_id", referenceID)

	servicehandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := servicehandler.QueuecallKickByReferenceID(c.Request.Context(), &a, referenceID)
	if err != nil {
		log.Errorf("Could not kick the queuecall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	c.JSON(200, res)
}
