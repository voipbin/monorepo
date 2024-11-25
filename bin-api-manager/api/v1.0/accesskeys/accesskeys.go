package accesskeys

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

// accesskeysPOST handles POST /accesskeys request.
// It creates a new agent.
//
//	@Summary		Create a new accesskey.
//	@Description	create a new accesskey
//	@Produce		json
//	@Param			agent	body		request.BodyAccesskeysPOST	true	"The accesskey detail"
//	@Success		200		{object}	accesskey.Accesskey
//	@Router			/v1.0/accesskeys [post]
func accesskeysPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "accesskeysPOST",
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

	var req request.BodyAccesskeysPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create
	res, err := serviceHandler.AccesskeyCreate(c.Request.Context(), &a, req.Name, req.Detail, req.Expire)
	if err != nil {
		log.Errorf("Could not create a accesskey. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// accesskeysGET handles GET /accesskeys request.
// It returns list of accesskeys of the given customer.

// @Summary		Get list of accesskeys
// @Description	get accesskeys of the customer
// @Produce		json
// @Param			page_size	query		int		false	"The size of results. Max 100"
// @Param			page_token	query		string	false	"The token. tm_create"
// @Success		200			{object}	response.BodyAccesskeysGET
// @Router			/v1.0/accesskeys [get]
func accesskeysGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "accesskeysGET",
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

	var requestParam request.ParamAccesskeysGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("accesskeysGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	tmps, err := serviceHandler.AccesskeyGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get calls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyAccesskeysGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// accesskeysIDGet handles GET /accesskeys/{accesskey-id} request.
// It returns detail accesskey info.
//
//	@Summary		Get detail caccesskeyall info.
//	@Description	Returns detail accesskey info of the given accesskey id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the accesskey"
//	@Success		200	{object}	accesskey.Accesskey
//	@Router			/v1.0/accesskeys/{accesskey-id} [get]
func accesskeysIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "accesskeysIDGet",
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

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.AccesskeyGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// accesskeysIDDelete handles DELETE /accesskeys/<accesskey-id> request.
// It deletes the call.
//
//	@Summary		Delete the accesskey
//	@Description	Delete the accesskey of the given id
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the accesskey"
//	@Success		200	{object}	accesskey.Accesskey
//	@Router			/v1.0/accesskeys/{accesskey-id} [delete]
func accesskeysIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "accesskeysIDDelete",
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

	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.AccesskeyDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// accesskeysIDPUT handles PUT /campaigns/{id} request.
// It updates a exist campaign info with the given campaign info.
// And returns updated campaign info if it succeed.
//
//	@Summary		Update a campaign and reuturns updated campaign info.
//	@Description	Update a campaign and returns detail updated campaign info.
//	@Produce		json
//	@Param			id			query		string						true	"The campaign's id"
//	@Param			update_info	body		request.BodyCampaignsIDPUT	true	"The update info"
//	@Success		200			{object}	campaign.Campaign
//	@Router			/v1.0/campaigns/{id} [put]
func accesskeysIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "accesskeysIDPUT",
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
	log = log.WithField("campaign_id", id)

	var req request.BodyAccesskeysIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing campaignsIDPUT.")

	// update a campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.AccesskeyUpdate(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
