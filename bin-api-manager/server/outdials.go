package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostOutdials(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostOutdials",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostOutdialsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	campaignID := uuid.FromStringOrNil(req.CampaignId)
	// OpenAPI marks campaign_id/name/detail/data as required, but BindJSON
	// doesn't enforce presence on string fields (an omitted field just binds
	// to the zero value). Guard explicitly so the spec's contract is
	// actually honored.
	if campaignID == uuid.Nil {
		log.Error("campaign_id is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "campaign_id is required."))
		return
	}
	if req.Name == "" {
		log.Error("name is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "name is required."))
		return
	}
	if req.Detail == "" {
		log.Error("detail is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "detail is required."))
		return
	}
	if req.Data == "" {
		log.Error("data is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "data is required."))
		return
	}

	res, err := h.serviceHandler.OutdialCreate(c.Request.Context(), a, campaignID, req.Name, req.Detail, req.Data)
	if err != nil {
		log.Errorf("Could not create a outdial. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetOutdials(c *gin.Context, params openapi_server.GetOutdialsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetOutdials",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)
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

	tmps, err := h.serviceHandler.OutdialGetsByCustomerID(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a outdial list. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
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

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.OutdialGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a outdial. err: %v", err)
		abortWithServiceError(c, err)
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

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.OutdialDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete the outdial. err: %v", err)
		abortWithServiceError(c, err)
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

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutOutdialsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.OutdialUpdateBasicInfo(c.Request.Context(), a, target, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		abortWithServiceError(c, err)
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

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutOutdialsIdCampaignIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	campaignID := uuid.FromStringOrNil(req.CampaignId)

	res, err := h.serviceHandler.OutdialUpdateCampaignID(c.Request.Context(), a, target, campaignID)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		abortWithServiceError(c, err)
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

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutOutdialsIdDataJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.OutdialUpdateData(c.Request.Context(), a, target, req.Data)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostOutdialsIdTargets(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostOutdialsIdTargets",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PostOutdialsIdTargetsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	destination0 := ConvertCommonAddress(req.Destination0)
	destination1 := ConvertCommonAddress(req.Destination1)
	destination2 := ConvertCommonAddress(req.Destination2)
	destination3 := ConvertCommonAddress(req.Destination3)
	destination4 := ConvertCommonAddress(req.Destination4)

	// OpenAPI marks name/detail/data and all 5 destination_N fields as
	// required, but BindJSON doesn't enforce presence on a struct field --
	// an omitted destination_N silently binds to a zero-value CommonAddress.
	// name/detail/data are enforced as the spec states (must be non-empty).
	// For the 5 destination_N fields, this deliberately enforces a looser
	// rule than the spec's literal "all 5 required" -- only that at least
	// one destination has a target -- since the domain model treats each
	// destination_N as an independent, nullable retry slot (see
	// bin-outdial-manager's outdial target dbhandler) and a target-less
	// destination has no reachable endpoint anyway. The spec's `required`
	// list overstates the actual constraint; tracked in VOIP-1308 to relax
	// the spec once the oapi-codegen regeneration blocker is resolved.
	if req.Name == "" {
		log.Error("name is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "name is required."))
		return
	}
	if req.Detail == "" {
		log.Error("detail is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "detail is required."))
		return
	}
	if req.Data == "" {
		log.Error("data is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "data is required."))
		return
	}
	if destination0.Target == "" && destination1.Target == "" && destination2.Target == "" && destination3.Target == "" && destination4.Target == "" {
		log.Error("At least one destination_N target is required.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "At least one destination_N target is required."))
		return
	}

	res, err := h.serviceHandler.OutdialtargetCreate(c.Request.Context(), a, target, req.Name, req.Detail, req.Data, &destination0, &destination1, &destination2, &destination3, &destination4)
	if err != nil {
		log.Errorf("Could not update the outdial. err: %v", err)
		abortWithServiceError(c, err)
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

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	targetID := uuid.FromStringOrNil(targetId)
	if targetID == uuid.Nil {
		log.Error("Could not parse the target_id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided target_id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.OutdialtargetGet(c.Request.Context(), a, target, targetID)
	if err != nil {
		log.Errorf("Could not get the outdial target. err: %v", err)
		abortWithServiceError(c, err)
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

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	targetID := uuid.FromStringOrNil(targetId)
	if targetID == uuid.Nil {
		log.Error("Could not parse the target_id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided target_id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.OutdialtargetDelete(c.Request.Context(), a, target, targetID)
	if err != nil {
		log.Errorf("Could not delete the outdial target. err: %v", err)
		abortWithServiceError(c, err)
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

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
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

	tmps, err := h.serviceHandler.OutdialtargetGetsByOutdialID(c.Request.Context(), a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a outdial list. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}
