package campaigncalls

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	_ "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall" // for sweg use.

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// campaigncallsGET handles GET /campaigncalls request.
// It returns list of campaigncalls of the given customer.

//	@Summary		Get list of calls
//	@Description	get calls of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyCallsGET
//	@Router			/v1.0/campaigncalls [get]
func campaigncallsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaigncallsGET",
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

	var requestParam request.ParamCampaigncallsGET
	if err := c.BindQuery(&requestParam); err != nil {
		log.Errorf("Could not parse the reqeust parameter. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get tmps
	tmps, err := serviceHandler.CampaigncallGets(c.Request.Context(), &a, pageSize, requestParam.PageToken)
	if err != nil {
		logrus.Errorf("Could not get campaigncalls info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyCampaigncallsGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// campaigncallsIDGET handles GET /campaigncalls/{id} request.
// It returns detail campaigncall info.
//	@Summary		Returns detail campaigncall info.
//	@Description	Returns detail campaigns info of the given campaigncall id.
//	@Produce		json
//	@Param			id	path		string	true	"The ID of the campaigncall"
//	@Success		200	{object}	campaigncall.WebhookMessage
//	@Router			/v1.0/campaigncalls/{id} [get]
func campaigncallsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaigncallsIDGET",
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
	res, err := serviceHandler.CampaigncallGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// campaigncallsIDDELETE handles DELETE /campaigncalls/{id} request.
// It finishs a campaigncall.
//	@Summary		Finish a existing campaign.
//	@Description	Delete a existing campaign.
//	@Produce		json
//	@Param			id	query	string	true	"The campaign's id"
//	@Success		200
//	@Router			/v1.0/campaigncalls/{id} [delete]
func campaigncallsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaigncallsIDDELETE",
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
	log.Debug("Executing campaigncallsIDDELETE.")

	// delete an campaign
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.CampaigncallDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the campaigncall. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
