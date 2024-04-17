package outdials

import (
	_ "monorepo/bin-outdial-manager/models/outdial" // for swag use

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// outdialsPOST handles POST /outdials request.
// It creates a new outdial with the given info and returns created outdial info.
//
//	@Summary		Create a new outdial and returns detail created outdial info.
//	@Description	Create a new outdial and returns detail created outdial info.
//	@Produce		json
//	@Param			outdial	body		request.BodyOutdialsPOST	true	"outdial info."
//	@Success		200		{object}	outdial.WebhookMessage
//	@Router			/v1.0/outdials [post]
func outdialsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsPOST",
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

	var req request.BodyOutdialsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing outdialsPOST.")

	// create a outdial
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialCreate(c.Request.Context(), &a, req.CampaignID, req.Name, req.Detail, req.Data)
	if err != nil {
		log.Errorf("Could not create a outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsGET handles GET /outdials request.
// It gets a list of outdials with the given info.
//
//	@Summary		Gets a list of outdials.
//	@Description	Gets a list of outdials
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyOutdialsGET
//	@Router			/v1.0/outdials [get]
func outdialsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsGET",
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

	var req request.ParamOutdialsGET
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
	log.Debugf("outdialsGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get outdial
	outdials, err := serviceHandler.OutdialGetsByCustomerID(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a outdial list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(outdials) > 0 {
		nextToken = outdials[len(outdials)-1].TMCreate
	}
	res := response.BodyOutdialsGET{
		Result: outdials,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// outdialsIDGET handles GET /outdials/{id} request.
// It returns detail outdials info.
//
//	@Summary		Returns detail outdials info.
//	@Description	Returns detail outdials info of the given outdials id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the outdials"
//	@Success		200	{object}	outdial.Outdial
//	@Router			/v1.0/outdials/{id} [get]
func outdialsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDGET",
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
	log = log.WithField("outdial_id", id)
	log.Debug("Executing outdialsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsIDDELETE handles DELETE /outdials/{id} request.
// It deletes a exist outdial info.
//
//	@Summary		Delete a existing outdial.
//	@Description	Delete a existing outdial.
//	@Produce		json
//	@Param			id	query	string	true	"The outdial's id"
//	@Success		200
//	@Router			/v1.0/outdials/{id} [delete]
func outdialsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDDELETE",
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
	log = log.WithField("outdial_id", id)
	log.Debug("Executing outdialsIDDELETE.")

	// delete an outdial
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsIDPUT handles PUT /outdials/{id} request.
// It updates a exist outdial info with the given outdial info.
// And returns updated outdial info if it succeed.
//
//	@Summary		Update a outdial and reuturns updated outdial info.
//	@Description	Update a outdial and returns detail updated outdial info.
//	@Produce		json
//	@Param			id			query		string						true	"The outdial's id"
//	@Param			update_info	body		request.BodyOutdialsIDPUT	true	"The update info"
//	@Success		200			{object}	outdial.Outdial
//	@Router			/v1.0/outdials/{id} [put]
func outdialsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDPUT",
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
	log = log.WithField("outdial_id", id)

	var req request.BodyOutdialsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing outdialsIDPUT.")

	// update a outdial
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialUpdateBasicInfo(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsIDCampaignIDPUT handles PUT /outdials/{id}/campaign_id request.
// It updates a exist outdial's campaign_id info with the given outdial info.
// And returns updated outdial's campaign_id info if it succeed.
//
//	@Summary		Update a outdial's campaign_id and reuturns updated outdial info.
//	@Description	Update a outdial's campaign_id and returns detail updated outdial info.
//	@Produce		json
//	@Param			id			query		string						true	"The outdial's id"
//	@Param			update_info	body		request.BodyOutdialsIDPUT	true	"The update info"
//	@Success		200			{object}	outdial.Outdial
//	@Router			/v1.0/outdials/{id}/campaign_id [put]
func outdialsIDCampaignIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDCampaignIDPUT",
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
	log = log.WithField("outdial_id", id)

	var req request.BodyOutdialsIDCampaignIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing outdialsIDPUT.")

	// update a outdial
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialUpdateCampaignID(c.Request.Context(), &a, id, req.CampaignID)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsIDDataPUT handles PUT /outdials/{id}/data request.
// It updates a exist outdial's data info with the given outdial info.
// And returns updated outdial's data info if it succeed.
//
//	@Summary		Update a outdial's data and reuturns updated outdial info.
//	@Description	Update a outdial's data and returns detail updated outdial info.
//	@Produce		json
//	@Param			id			query		string							true	"The outdial's id"
//	@Param			update_info	body		request.BodyOutdialsIDDataPUT	true	"The update info"
//	@Success		200			{object}	outdial.Outdial
//	@Router			/v1.0/outdials/{id}/data [put]
func outdialsIDDataPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDCampaignIDPUT",
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
	log = log.WithField("outdial_id", id)

	var req request.BodyOutdialsIDDataPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing outdialsIDPUT.")

	// update a outdial
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialUpdateData(c.Request.Context(), &a, id, req.Data)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsIDTargetsPOST handles POST /outdials/{id}/targets request.
// It creates a new outdial's target info with the given target info.
// And returns created outdial target's data info if it succeed.
//
//	@Summary		Create a new outdialtarget's data and reuturns updated outdial info.
//	@Description	Update a outdial's data and returns detail updated outdial info.
//	@Produce		json
//	@Param			id			query		string							true	"The outdial's id"
//	@Param			update_info	body		request.BodyOutdialsIDDataPUT	true	"The update info"
//	@Success		200			{object}	outdial.Outdial
//	@Router			/v1.0/outdials/{id}/targets [post]
func outdialsIDTargetsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDTargetsPOST",
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
	log = log.WithField("outdial_id", id)

	var req request.BodyOutdialsIDTargetsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing outdialsIDTargetsPOST.")

	// create a target
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialtargetCreate(c.Request.Context(), &a, id, req.Name, req.Detail, req.Data, req.Destination0, req.Destination1, req.Destination2, req.Destination3, req.Destination4)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsIDTargetsIDGET handles GET /outdials/{id}/targets/{target_id} request.
// It gets a exist outdialtarget info.
//
//	@Summary		Get a existing outdialtarget.
//	@Description	Get a existing outdialtarget.
//	@Produce		json
//	@Param			id			query	string	true	"The outdial's id"
//	@Param			target_id	query	string	true	"The outdialtarget's id"
//	@Success		200
//	@Router			/v1.0/outdials/{id}/targets/{target_id} [get]
func outdialsIDTargetsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDTargetsIDGET",
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
	targetID := uuid.FromStringOrNil(c.Params.ByName("target_id"))
	log = log.WithFields(logrus.Fields{
		"outdial_id":       id,
		"outdialtarget_id": targetID,
	})
	log.Debug("Executing outdialsIDTargetsIDGET.")

	// get an outdial target
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialtargetGet(c.Request.Context(), &a, id, targetID)
	if err != nil {
		log.Errorf("Could not get the outdial target. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsIDTargetsIDDELETE handles DELETE /outdials/{id}/targets/{target_id} request.
// It deletes a exist outdialtarget info.
//
//	@Summary		Delete a existing outdialtarget.
//	@Description	Delete a existing outdialtarget.
//	@Produce		json
//	@Param			id			query	string	true	"The outdial's id"
//	@Param			target_id	query	string	true	"The outdialtarget's id"
//	@Success		200
//	@Router			/v1.0/outdials/{id}/targets/{target_id} [delete]
func outdialsIDTargetsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDTargetsIDDELETE",
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
	targetID := uuid.FromStringOrNil(c.Params.ByName("target_id"))
	log = log.WithFields(logrus.Fields{
		"outdial_id":       id,
		"outdialtarget_id": targetID,
	})
	log.Debug("Executing outdialsIDTargetsIDDELETE.")

	// delete an outdial target
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.OutdialtargetDelete(c.Request.Context(), &a, id, targetID)
	if err != nil {
		log.Errorf("Could not delete the outdial target. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// outdialsIDTargetsGET handles GET /outdials/{id}/targets request.
// It gets a list of outdialtargets.
//
//	@Summary		Get a list of outdialtargets.
//	@Description	Get a list of outdialtargets.
//	@Produce		json
//	@Param			id	query	string	true	"The outdial's id"
//	@Success		200
//	@Router			/v1.0/outdials/{id}/targets [get]
func outdialsIDTargetsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "outdialsIDTargetsGET",
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
	log = log.WithFields(logrus.Fields{
		"outdial_id": id,
	})
	log.Debug("Executing outdialsIDTargetsGET.")

	var req request.ParamOutdialsIDTargetsGET
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
	log.Debugf("outdialsGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get outdial
	outdialtargets, err := serviceHandler.OutdialtargetGetsByOutdialID(c.Request.Context(), &a, id, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a outdial list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(outdialtargets) > 0 {
		nextToken = outdialtargets[len(outdialtargets)-1].TMCreate
	}
	res := response.BodyOutdialsIDTargetsGET{
		Result: outdialtargets,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}
