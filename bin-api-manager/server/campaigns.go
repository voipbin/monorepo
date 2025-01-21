package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	cmcampaign "monorepo/bin-campaign-manager/models/campaign"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostCampaigns(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCampaigns",
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

	var req openapi_server.PostCampaignsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	actions := []fmaction.Action{}
	for _, v := range req.Actions {
		tmp := ConvertFlowManagerAction(v)
		actions = append(actions, tmp)
	}

	outplanID := uuid.FromStringOrNil(req.OutplanId)
	outdialID := uuid.FromStringOrNil(req.OutdialId)
	queueID := uuid.FromStringOrNil(req.QueueId)
	nextCampaignID := uuid.FromStringOrNil(req.NextCampaignId)

	res, err := h.serviceHandler.CampaignCreate(c.Request.Context(), &a, req.Name, req.Detail, cmcampaign.Type(req.Type), req.ServiceLevel, cmcampaign.EndHandle(req.EndHandle), actions, outplanID, outdialID, queueID, nextCampaignID)
	if err != nil {
		log.Errorf("Could not create a campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetCampaigns(c *gin.Context, params openapi_server.GetCampaignsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCampaigns",
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

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.CampaignGetsByCustomerID(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a campaign list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) GetCampaignsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCampaignsId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CampaignGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteCampaignsId(c *gin.Context, id string) {
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CampaignDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCampaignsId(c *gin.Context, id string) {
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCampaignsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CampaignUpdateBasicInfo(c.Request.Context(), &a, target, req.Name, req.Detail, cmcampaign.Type(req.Type), req.ServiceLevel, cmcampaign.EndHandle(req.EndHandle))
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCampaignsIdStatus(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCampaignsIdStatus",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCampaignsIdStatusJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CampaignUpdateStatus(c.Request.Context(), &a, target, cmcampaign.Status(req.Status))
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCampaignsIdServiceLevel(c *gin.Context, id string) {
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCampaignsIdServiceLevelJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CampaignUpdateServiceLevel(c.Request.Context(), &a, target, req.ServiceLevel)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCampaignsIdActions(c *gin.Context, id string) {
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCampaignsIdActionsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	actions := []fmaction.Action{}
	for _, v := range req.Actions {
		tmp := ConvertFlowManagerAction(v)
		actions = append(actions, tmp)
	}

	res, err := h.serviceHandler.CampaignUpdateActions(c.Request.Context(), &a, target, actions)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCampaignsIdResourceInfo(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCampaignsIdResourceInfo",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCampaignsIdResourceInfoJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	outplanID := uuid.FromStringOrNil(req.OutplanId)
	outdialID := uuid.FromStringOrNil(req.OutdialId)
	queueID := uuid.FromStringOrNil(req.QueueId)
	nextCampaignID := uuid.FromStringOrNil(req.NextCampaignId)

	res, err := h.serviceHandler.CampaignUpdateResourceInfo(c.Request.Context(), &a, target, outplanID, outdialID, queueID, nextCampaignID)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutCampaignsIdNextCampaignId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutCampaignsIdNextCampaignId",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutCampaignsIdNextCampaignIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextCampaignID := uuid.FromStringOrNil(req.NextCampaignId)

	res, err := h.serviceHandler.CampaignUpdateNextCampaignID(c.Request.Context(), &a, target, nextCampaignID)
	if err != nil {
		log.Errorf("Could not update the campaign. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetCampaignsIdCampaigncalls(c *gin.Context, id string, params openapi_server.GetCampaignsIdCampaigncallsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCampaignsIdCampaigncalls",
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.CampaigncallGetsByCampaignID(c.Request.Context(), &a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a campaign list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}
