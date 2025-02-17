package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostOutdials(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostOutdials",
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

	var req openapi_server.PostOutdialsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	campaignID := uuid.FromStringOrNil(req.CampaignId)

	res, err := h.serviceHandler.OutdialCreate(c.Request.Context(), &a, campaignID, req.Name, req.Detail, req.Data)
	if err != nil {
		log.Errorf("Could not create a outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetOutdials(c *gin.Context, params openapi_server.GetOutdialsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutdials",
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

	tmps, err := h.serviceHandler.OutdialGetsByCustomerID(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a outdial list. err: %v", err)
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

func (h *server) GetOutdialsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutdialsId",
		"request_address": c.ClientIP,
		"outdial_id":      id,
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

	res, err := h.serviceHandler.OutdialGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteOutdialsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteOutdialsId",
		"request_address": c.ClientIP,
		"outdial_id":      id,
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

	res, err := h.serviceHandler.OutdialDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutOutdialsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutOutdialsId",
		"request_address": c.ClientIP,
		"outdial_id":      id,
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

	var req openapi_server.PutOutdialsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.OutdialUpdateBasicInfo(c.Request.Context(), &a, target, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutOutdialsIdCampaignId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutOutdialsIdCampaignId",
		"request_address": c.ClientIP,
		"outdial_id":      id,
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

	var req openapi_server.PutOutdialsIdCampaignIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	campaignID := uuid.FromStringOrNil(req.CampaignId)

	res, err := h.serviceHandler.OutdialUpdateCampaignID(c.Request.Context(), &a, target, campaignID)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutOutdialsIdData(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutOutdialsIdData",
		"request_address": c.ClientIP,
		"outdial_id":      id,
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

	var req openapi_server.PutOutdialsIdDataJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.OutdialUpdateData(c.Request.Context(), &a, target, req.Data)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostOutdialsIdTargets(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostOutdialsIdTargets",
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

	var req openapi_server.PostOutdialsIdTargetsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	destination0 := ConvertCommonAddress(req.Destination0)
	destination1 := ConvertCommonAddress(req.Destination1)
	destination2 := ConvertCommonAddress(req.Destination2)
	destination3 := ConvertCommonAddress(req.Destination3)
	destination4 := ConvertCommonAddress(req.Destination4)

	res, err := h.serviceHandler.OutdialtargetCreate(c.Request.Context(), &a, target, req.Name, req.Detail, req.Data, &destination0, &destination1, &destination2, &destination3, &destination4)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetOutdialsIdTargetsTargetId(c *gin.Context, id string, targetId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutdialsIdTargetsTargetId",
		"request_address": c.ClientIP,
		"outdial_id":      id,
		"target_id":       targetId,
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

	targetID := uuid.FromStringOrNil(targetId)

	res, err := h.serviceHandler.OutdialtargetGet(c.Request.Context(), &a, target, targetID)
	if err != nil {
		log.Errorf("Could not get the outdial target. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteOutdialsIdTargetsTargetId(c *gin.Context, id string, targetId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteOutdialsIdTargetsTargetId",
		"request_address": c.ClientIP,
		"outdial_id":      id,
		"target_id":       targetId,
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

	targetID := uuid.FromStringOrNil(targetId)

	res, err := h.serviceHandler.OutdialtargetDelete(c.Request.Context(), &a, target, targetID)
	if err != nil {
		log.Errorf("Could not delete the outdial target. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetOutdialsIdTargets(c *gin.Context, id string, params openapi_server.GetOutdialsIdTargetsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutdialsIdTargets",
		"request_address": c.ClientIP,
		"outdial_id":      id,
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

	tmps, err := h.serviceHandler.OutdialtargetGetsByOutdialID(c.Request.Context(), &a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a outdial list. err: %v", err)
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
