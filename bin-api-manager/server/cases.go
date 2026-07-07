package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GetCases handles GET /cases
func (h *server) GetCases(c *gin.Context, params openapi_server.GetCasesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCases",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize == 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = string(*params.PageToken)
	}

	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}

	ownerType := ""
	if params.OwnerType != nil {
		ownerType = *params.OwnerType
	}

	ownerID := uuid.Nil
	if params.OwnerId != nil {
		ownerID = uuid.UUID(*params.OwnerId)
	}

	// targetCustomerID is always uuid.Nil here -- this endpoint has no
	// client-supplied customer_id filter, so CaseList always resolves to
	// the authenticated caller's own a.CustomerID (see CaseList's
	// targetCustomerID == uuid.Nil default). customer_id is never taken
	// from client input.
	items, nextToken, err := h.serviceHandler.CaseList(c.Request.Context(), a, uuid.Nil, pageSize, pageToken, status, ownerType, ownerID)
	if err != nil {
		log.Errorf("Could not list cases. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}

// GetCasesUnresolved handles GET /cases/unresolved
func (h *server) GetCasesUnresolved(c *gin.Context, params openapi_server.GetCasesUnresolvedParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCasesUnresolved",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize == 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = string(*params.PageToken)
	}

	items, nextToken, err := h.serviceHandler.CaseListUnresolved(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not list unresolved cases. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}

// GetCasesId handles GET /cases/{id}
func (h *server) GetCasesId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetCasesId",
		"request_address": c.ClientIP(),
		"id":              id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	caseID := uuid.UUID(id)

	res, err := h.serviceHandler.CaseGet(c.Request.Context(), a, caseID)
	if err != nil {
		log.Errorf("Could not get case. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PostCasesIdClose handles POST /cases/{id}/close
func (h *server) PostCasesIdClose(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCasesIdClose",
		"request_address": c.ClientIP(),
		"id":              id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	var req openapi_server.PostCasesIdCloseJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	caseID := uuid.UUID(id)
	closedByID := uuid.UUID(req.ClosedById)

	res, err := h.serviceHandler.CaseClose(c.Request.Context(), a, caseID, closedByID)
	if err != nil {
		log.Errorf("Could not close case. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PostCasesIdContinue handles POST /cases/{id}/continue
func (h *server) PostCasesIdContinue(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCasesIdContinue",
		"request_address": c.ClientIP(),
		"id":              id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	caseID := uuid.UUID(id)

	res, err := h.serviceHandler.CaseContinue(c.Request.Context(), a, caseID)
	if err != nil {
		log.Errorf("Could not continue case. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
