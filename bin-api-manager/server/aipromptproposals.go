package server

import (
	amaipromptproposal "monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) PostAipromptproposals(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAipromptproposals",
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

	var req openapi_server.PostAipromptproposalsJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	aiID := uuid.FromStringOrNil(req.AiId)
	if aiID == uuid.Nil {
		log.Errorf("Could not parse the ai_id. ai_id: %s", req.AiId)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided ai_id is not a valid UUID."))
		return
	}

	auditIDs := make([]uuid.UUID, 0, len(req.AuditIds))
	for _, s := range req.AuditIds {
		auditID := uuid.FromStringOrNil(s)
		if auditID == uuid.Nil {
			log.Errorf("Could not parse an audit_id. audit_id: %s", s)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided audit_id is not a valid UUID."))
			return
		}
		auditIDs = append(auditIDs, auditID)
	}

	language := ""
	if req.Language != nil {
		language = *req.Language
	}

	tmp, err := h.serviceHandler.AIPromptProposalCreate(c.Request.Context(), a, aiID, auditIDs, language)
	if err != nil {
		log.Errorf("Could not create AI prompt proposal. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(202, tmp)
}

func (h *server) GetAipromptproposals(c *gin.Context, params openapi_server.GetAipromptproposalsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAipromptproposals",
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

	aiID := uuid.Nil
	if params.AiId != nil {
		aiID = uuid.UUID(*params.AiId)
	}

	status := amaipromptproposal.Status("")
	if params.Status != nil {
		status = amaipromptproposal.Status(*params.Status)
	}

	tmps, err := h.serviceHandler.AIPromptProposalGetsByCustomerID(c.Request.Context(), a, pageSize, pageToken, aiID, status)
	if err != nil {
		log.Errorf("Could not get AI prompt proposal list. err: %v", err)
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

func (h *server) GetAipromptproposalsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "GetAipromptproposalsId",
		"request_address":        c.ClientIP,
		"aipromptproposal_id":    id,
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

	res, err := h.serviceHandler.AIPromptProposalGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get AI prompt proposal. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostAipromptproposalsIdAccept(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "PostAipromptproposalsIdAccept",
		"request_address":     c.ClientIP,
		"aipromptproposal_id": id,
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

	res, err := h.serviceHandler.AIPromptProposalAccept(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not accept AI prompt proposal. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostAipromptproposalsIdReject(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "PostAipromptproposalsIdReject",
		"request_address":     c.ClientIP,
		"aipromptproposal_id": id,
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

	res, err := h.serviceHandler.AIPromptProposalReject(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not reject AI prompt proposal. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteAipromptproposalsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "DeleteAipromptproposalsId",
		"request_address":     c.ClientIP,
		"aipromptproposal_id": id,
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

	res, err := h.serviceHandler.AIPromptProposalDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete AI prompt proposal. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
