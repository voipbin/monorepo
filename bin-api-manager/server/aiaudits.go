package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) PostAiaudits(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAiaudits",
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

	var req openapi_server.PostAiauditsJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	aicallID := uuid.FromStringOrNil(req.AicallId)

	language := ""
	if req.Language != nil {
		language = *req.Language
	}

	tmps, err := h.serviceHandler.AIAuditCreate(c.Request.Context(), a, aicallID, language)
	if err != nil {
		log.Errorf("Could not create AI audits. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(202, gin.H{"result": tmps})
}

func (h *server) GetAiaudits(c *gin.Context, params openapi_server.GetAiauditsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAiaudits",
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

	aicallID := uuid.Nil
	if params.AicallId != nil {
		aicallID = uuid.UUID(*params.AicallId)
	}

	aiID := uuid.Nil
	if params.AiId != nil {
		aiID = uuid.UUID(*params.AiId)
	}

	tmps, err := h.serviceHandler.AIAuditGetsByCustomerID(c.Request.Context(), a, pageSize, pageToken, aicallID, aiID)
	if err != nil {
		log.Errorf("Could not get AI audit list. err: %v", err)
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

func (h *server) GetAiauditsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAiauditsId",
		"request_address": c.ClientIP,
		"aiaudit_id":      id,
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

	target := uuid.UUID(id)

	res, err := h.serviceHandler.AIAuditGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get AI audit. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteAiauditsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteAiauditsId",
		"request_address": c.ClientIP,
		"aiaudit_id":      id,
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

	target := uuid.UUID(id)

	res, err := h.serviceHandler.AIAuditDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete AI audit. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
