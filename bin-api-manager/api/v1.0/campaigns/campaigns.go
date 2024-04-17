package campaigns

import (
	_ "monorepo/bin-campaign-manager/models/campaign" // for swag use

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// campaignsPOST handles POST /campaigns request.
// It creates a new campaign with the given info and returns created campaign info.
//
//	@Summary		Create a new campaign and returns detail created campaign info.
//	@Description	Create a new campaign and returns detail created campaign info.
//	@Produce		json
//	@Param			campaign	body		request.BodyCampaignsPOST	true	"campaign info."
//	@Success		200			{object}	campaign.WebhookMessage
//	@Router			/v1.0/campaigns [post]
func campaignsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsPOST",
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

	var req request.BodyCampaignsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing campaignsPOST.")

	// create a campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignCreate(c.Request.Context(), &a, req.Name, req.Detail, req.Type, req.ServiceLevel, req.EndHandle, req.Actions, req.OutplanID, req.OutdialID, req.QueueID, req.NextCampaignID)
	if err != nil {
		log.Errorf("Could not create a campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsGET handles GET /campaigns request.
// It gets a list of campaigns with the given info.
//
//	@Summary		Gets a list of campaigns.
//	@Description	Gets a list of campaigns
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyCampaignsGET
//	@Router			/v1.0/campaigns [get]
func campaignsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsGET",
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

	var req request.ParamCampaignsGET
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
	log.Debugf("campaignsGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get campaign
	campaigns, err := serviceHandler.CampaignGetsByCustomerID(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a campaign list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(campaigns) > 0 {
		nextToken = campaigns[len(campaigns)-1].TMCreate
	}
	res := response.BodyCampaignsGET{
		Result: campaigns,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// campaignsIDGET handles GET /campaigns/{id} request.
// It returns detail campaigns info.
//
//	@Summary		Returns detail campaigns info.
//	@Description	Returns detail campaigns info of the given campaigns id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the campaigns"
//	@Success		200	{object}	campaign.Campaign
//	@Router			/v1.0/campaigns/{id} [get]
func campaignsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDGET",
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
	log.Debug("Executing campaignsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsIDDELETE handles DELETE /campaigns/{id} request.
// It deletes a exist campaign info.
//
//	@Summary		Delete a existing campaign.
//	@Description	Delete a existing campaign.
//	@Produce		json
//	@Param			id	query	string	true	"The campaign's id"
//	@Success		200
//	@Router			/v1.0/campaigns/{id} [delete]
func campaignsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDDELETE",
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
	log.Debug("Executing campaignsIDDELETE.")

	// delete an campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsIDPUT handles PUT /campaigns/{id} request.
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
func campaignsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDPUT",
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

	var req request.BodyCampaignsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing campaignsIDPUT.")

	// update a campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignUpdateBasicInfo(c.Request.Context(), &a, id, req.Name, req.Detail, req.Type, req.ServiceLevel, req.EndHandle)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsIDStatusPUT handles PUT /campaigns/{id}/dial_info request.
// It updates a exist campaign info with the given campaign info.
// And returns updated campaign info if it succeed.
//
//	@Summary		Update a campaign and reuturns updated campaign info.
//	@Description	Update a campaign and returns detail updated campaign info.
//	@Produce		json
//	@Param			id			query		string								true	"The campaign's id"
//	@Param			update_info	body		request.BodyCampaignsIDStatusPUT	true	"The update info"
//	@Success		200			{object}	campaign.Campaign
//	@Router			/v1.0/campaigns/{id}/status [put]
func campaignsIDStatusPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDStatusPUT",
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

	var req request.BodyCampaignsIDStatusPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing campaignsIDDialInfoPUT.")

	// update a campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignUpdateStatus(c.Request.Context(), &a, id, req.Status)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsIDServiceLevelPUT handles PUT /campaigns/{id}/service_level request.
// It updates a exist campaign info with the given campaign info.
// And returns updated campaign info if it succeed.
//
//	@Summary		Update a campaign and reuturns updated campaign info.
//	@Description	Update a campaign and returns detail updated campaign info.
//	@Produce		json
//	@Param			id			query		string								true	"The campaign's id"
//	@Param			update_info	body		request.BodyCampaignsIDStatusPUT	true	"The update info"
//	@Success		200			{object}	campaign.Campaign
//	@Router			/v1.0/campaigns/{id}/service_level [put]
func campaignsIDServiceLevelPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDServiceLevelPUT",
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

	var req request.BodyCampaignsIDServiceLevelPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing campaignsIDServiceLevelPUT.")

	// update a campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignUpdateServiceLevel(c.Request.Context(), &a, id, req.ServiceLevel)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsIDActionsPUT handles PUT /campaigns/{id}/service_level request.
// It updates a exist campaign info with the given campaign info.
// And returns updated campaign info if it succeed.
//
//	@Summary		Update a campaign and reuturns updated campaign info.
//	@Description	Update a campaign and returns detail updated campaign info.
//	@Produce		json
//	@Param			id			query		string								true	"The campaign's id"
//	@Param			update_info	body		request.BodyCampaignsIDStatusPUT	true	"The update info"
//	@Success		200			{object}	campaign.Campaign
//	@Router			/v1.0/campaigns/{id}/actions [put]
func campaignsIDActionsPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDActionsPUT",
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

	var req request.BodyCampaignsIDActionsPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing campaignsIDActionsPUT.")

	// update a campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignUpdateActions(c.Request.Context(), &a, id, req.Actions)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsIDResourceInfoPUT handles PUT /campaigns/{id}/resource_info request.
// It updates a exist campaign info with the given campaign info.
// And returns updated campaign info if it succeed.
//
//	@Summary		Update a campaign and reuturns updated campaign info.
//	@Description	Update a campaign and returns detail updated campaign info.
//	@Produce		json
//	@Param			id			query		string									true	"The campaign's id"
//	@Param			update_info	body		request.BodyCampaignsIDResourceInfoPUT	true	"The update info"
//	@Success		200			{object}	campaign.Campaign
//	@Router			/v1.0/campaigns/{id}/resource_info [put]
func campaignsIDResourceInfoPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDActionsPUT",
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

	var req request.BodyCampaignsIDResourceInfoPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing campaignsIDResourceInfoPUT.")

	// update a campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignUpdateResourceInfo(c.Request.Context(), &a, id, req.OutplanID, req.OutdialID, req.QueueID, req.NextCampaignID)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsIDResourceInfoPUT handles PUT /campaigns/{id}/resource_info request.
// It updates a exist campaign info with the given campaign info.
// And returns updated campaign info if it succeed.
//
//	@Summary		Update a campaign and reuturns updated campaign info.
//	@Description	Update a campaign and returns detail updated campaign info.
//	@Produce		json
//	@Param			id			query		string									true	"The campaign's id"
//	@Param			update_info	body		request.BodyCampaignsIDResourceInfoPUT	true	"The update info"
//	@Success		200			{object}	campaign.Campaign
//	@Router			/v1.0/campaigns/{id}/next_campaign_id [put]
func campaignsIDNextCampaignIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDNextCampaignIDPUT",
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

	var req request.BodyCampaignsIDNextCampaignIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing campaignsIDNextCampaignIDPUT.")

	// update a campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaignUpdateNextCampaignID(c.Request.Context(), &a, id, req.NextCampaignID)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaignsIDCampaigncallsGET handles GET /campaigns/{id}/campaigncalls request.
// It gets a list of campaigncalls with the given info.
//
//	@Summary		Gets a list of campaigns.
//	@Description	Gets a list of campaigns
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyCampaignsIDCampaigncallsGET
//	@Router			/v1.0/campaigns/{id}/campaigncalls [get]
func campaignsIDCampaigncallsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaignsIDCampaigncallsGET",
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
	log = log.WithField("campaign_id", id)

	log = log.WithFields(logrus.Fields{
		"campaign_id": id,
	})

	var req request.ParamCampaignsIDCampaigncallsGET
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
	log.Debugf("campaignsGET. Received request detail. page_size: %d, page_token: %s", pageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get campaigncalls
	campaigncalls, err := serviceHandler.CampaigncallGetsByCampaignID(c.Request.Context(), &a, id, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a campaign list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(campaigncalls) > 0 {
		nextToken = campaigncalls[len(campaigncalls)-1].TMCreate
	}
	res := response.BodyCampaignsIDCampaigncallsGET{
		Result: campaigncalls,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}
