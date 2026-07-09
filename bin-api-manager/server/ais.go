package server

import (
	amai "monorepo/bin-ai-manager/models/ai"
	amtool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) PostAis(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAis",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	var req openapi_server.PostAisJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	// Convert tool names if provided
	var toolNames []amtool.ToolName
	if req.ToolNames != nil {
		toolNames = make([]amtool.ToolName, len(*req.ToolNames))
		for i, name := range *req.ToolNames {
			toolNames[i] = amtool.ToolName(name)
		}
	}

	var ragID uuid.UUID
	if req.RagId != nil {
		ragID = uuid.FromStringOrNil(*req.RagId)
	}

	var sttLanguage string
	if req.SttLanguage != nil {
		sttLanguage = *req.SttLanguage
	}

	autoAICallAuditEnabled := false
	if req.AutoAicallAuditEnabled != nil {
		autoAICallAuditEnabled = *req.AutoAicallAuditEnabled
	}

	var aiType amai.Type
	if req.Type != nil {
		aiType = amai.Type(*req.Type)
	}

	res, err := h.serviceHandler.AICreate(
		c.Request.Context(),
		a,
		req.Name,
		req.Detail,
		aiType,
		amai.EngineModel(req.EngineModel),
		req.Parameter,
		req.EngineKey,
		ragID,
		req.InitPrompt,
		amai.TTSType(req.TtsType),
		req.TtsVoiceId,
		amai.STTType(req.SttType),
		sttLanguage,
		toolNames,
		autoAICallAuditEnabled,
	)
	if err != nil {
		log.Errorf("Could not create a AI. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetAis(c *gin.Context, params openapi_server.GetAisParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAis",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
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

	tmps, err := h.serviceHandler.AIGetsByCustomerID(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a AI list. err: %v", err)
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

func (h *server) GetAisId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAisId",
		"request_address": c.ClientIP,
		"ai_id":           id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.AIGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get an AI. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteAisId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteAisId",
		"request_address": c.ClientIP,
		"ai_id":           id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.AIDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete the ai. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutAisId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutAisId",
		"request_address": c.ClientIP,
		"ai_id":           id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutAisIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	// Convert tool names if provided
	var toolNames []amtool.ToolName
	if req.ToolNames != nil {
		toolNames = make([]amtool.ToolName, len(*req.ToolNames))
		for i, name := range *req.ToolNames {
			toolNames[i] = amtool.ToolName(name)
		}
	}

	var ragID uuid.UUID
	if req.RagId != nil {
		ragID = uuid.FromStringOrNil(*req.RagId)
	}

	var sttLanguage string
	if req.SttLanguage != nil {
		sttLanguage = *req.SttLanguage
	}

	autoAICallAuditEnabled := false
	if req.AutoAicallAuditEnabled != nil {
		autoAICallAuditEnabled = *req.AutoAicallAuditEnabled
	}

	var aiType amai.Type
	if req.Type != nil {
		aiType = amai.Type(*req.Type)
	}

	res, err := h.serviceHandler.AIUpdate(
		c.Request.Context(),
		a,
		target,
		req.Name,
		req.Detail,
		aiType,
		amai.EngineModel(req.EngineModel),
		req.Parameter,
		req.EngineKey,
		ragID,
		req.InitPrompt,
		amai.TTSType(req.TtsType),
		req.TtsVoiceId,
		amai.STTType(req.SttType),
		sttLanguage,
		toolNames,
		autoAICallAuditEnabled,
	)
	if err != nil {
		log.Errorf("Could not update the ai. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostAisIdDirectHashRegenerate(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAisIdDirectHashRegenerate",
		"request_address": c.ClientIP(),
		"ai_id":           id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	// Convert openapi_types.UUID to uuid.UUID
	aiID, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid AI ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.AIDirectHashRegenerate(c.Request.Context(), a, aiID)
	if err != nil {
		log.Errorf("Could not regenerate AI direct hash. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetAisIdParticipants(c *gin.Context, id openapi_types.UUID, params openapi_server.GetAisIdParticipantsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAisIdParticipants",
		"request_address": c.ClientIP(),
		"ai_id":           id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	aiID := uuid.UUID(id)

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

	tmps, err := h.serviceHandler.AIParticipantGets(c.Request.Context(), a, aiID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get AI participants list. err: %v", err)
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
