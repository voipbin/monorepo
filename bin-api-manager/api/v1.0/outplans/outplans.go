package outplans

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan" // for swag use

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// outplansPOST handles POST /outplans request.
// It creates a new outplan with the given info and returns created outplan info.
//	@Summary		Create a new outplan and returns detail created outplan info.
//	@Description	Create a new outplan and returns detail created outplan info.
//	@Produce		json
//	@Param			outplan	body		request.BodyOutplansPOST	true	"outplan info."
//	@Success		200		{object}	outplan.WebhookMessage
//	@Router			/v1.0/outplans [post]
func outplansPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outplansPOST",
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

	var req request.BodyOutplansPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing outplansPOST.")

	// create a outplan
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutplanCreate(c.Request.Context(), &a, req.Name, req.Detail, req.Source, req.DialTimeout, req.TryInterval, req.MaxTryCount0, req.MaxTryCount1, req.MaxTryCount2, req.MaxTryCount3, req.MaxTryCount4)
	if err != nil {
		log.Errorf("Could not create a outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outplansGET handles GET /outplans request.
// It gets a list of outplans with the given info.
//	@Summary		Gets a list of outplans.
//	@Description	Gets a list of outplans
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyOutplansGET
//	@Router			/v1.0/outplans [get]
func outplansGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outplansGET",
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

	var req request.ParamOutplansGET
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
	log.Debugf("outplansGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get outplan
	outplans, err := serviceHandler.OutplanGetsByCustomerID(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a outplan list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(outplans) > 0 {
		nextToken = outplans[len(outplans)-1].TMCreate
	}
	res := response.BodyOutplansGET{
		Result: outplans,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// outplansIDGET handles GET /outplans/{id} request.
// It returns detail outplans info.
//	@Summary		Returns detail outplans info.
//	@Description	Returns detail outplans info of the given outplans id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the outplans"
//	@Success		200	{object}	outplan.Outplan
//	@Router			/v1.0/outplans/{id} [get]
func outplansIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outplansIDGET",
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
	log = log.WithField("outplan_id", id)
	log.Debug("Executing outplansIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutplanGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outplansIDDELETE handles DELETE /outplans/{id} request.
// It deletes a exist outplan info.
//	@Summary		Delete a existing outplan.
//	@Description	Delete a existing outplan.
//	@Produce		json
//	@Param			id	query	string	true	"The outplan's id"
//	@Success		200
//	@Router			/v1.0/outplans/{id} [delete]
func outplansIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outplansIDDELETE",
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
	log = log.WithField("outplan_id", id)
	log.Debug("Executing outplansIDDELETE.")

	// delete an outplan
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutplanDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outplansIDPUT handles PUT /outplans/{id} request.
// It updates a exist outplan info with the given outplan info.
// And returns updated outplan info if it succeed.
//	@Summary		Update a outplan and reuturns updated outplan info.
//	@Description	Update a outplan and returns detail updated outplan info.
//	@Produce		json
//	@Param			id			query		string						true	"The outplan's id"
//	@Param			update_info	body		request.BodyOutplansIDPUT	true	"The update info"
//	@Success		200			{object}	outplan.Outplan
//	@Router			/v1.0/outplans/{id} [put]
func outplansIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outplansIDPUT",
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
	log = log.WithField("outplan_id", id)

	var req request.BodyOutplansIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing outplansIDPUT.")

	// update a outplan
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutplanUpdateBasicInfo(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outplansIDDialInfoPUT handles PUT /outplans/{id}/dialinfo request.
// It updates a exist outplan info with the given outplan info.
// And returns updated outplan info if it succeed.
//	@Summary		Update a outplan and reuturns updated outplan info.
//	@Description	Update a outplan and returns detail updated outplan info.
//	@Produce		json
//	@Param			id			query		string								true	"The outplan's id"
//	@Param			update_info	body		request.BodyOutplansIDDialInfoPUT	true	"The update info"
//	@Success		200			{object}	outplan.Outplan
//	@Router			/v1.0/outplans/{id} [put]
func outplansIDDialInfoPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outplansIDDialInfoPUT",
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
	log = log.WithField("outplan_id", id)

	var req request.BodyOutplansIDDialInfoPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing outplansIDPUT.")

	// update a outplan
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutplanUpdateDialInfo(c.Request.Context(), &a, id, req.Source, req.DialTimeout, req.TryInterval, req.MaxTryCount0, req.MaxTryCount1, req.MaxTryCount2, req.MaxTryCount3, req.MaxTryCount4)
	if err != nil {
		log.Errorf("Could not update the outplan. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
