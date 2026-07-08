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

// GetContactCases handles GET /contact_cases
func (h *server) GetContactCases(c *gin.Context, params openapi_server.GetContactCasesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactCases",
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

// GetContactCasesUnresolved handles GET /contact_cases/unresolved
func (h *server) GetContactCasesUnresolved(c *gin.Context, params openapi_server.GetContactCasesUnresolvedParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactCasesUnresolved",
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

// GetContactCasesId handles GET /contact_cases/{id}
func (h *server) GetContactCasesId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactCasesId",
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

// PostContactCasesIdClose handles POST /contact_cases/{id}/close. closed_by_id is
// derived server-side from the authenticated caller's own agent identity
// (CaseClose internally uses a.AgentID()) -- there is no request body to
// bind, matching PostContactCasesIdContinue's pattern below, so the
// closing-agent attribution the platform treats as a hard invariant
// (design §5.3) cannot be forged via a client-supplied agent_id.
func (h *server) PostContactCasesIdClose(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactCasesIdClose",
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

	res, err := h.serviceHandler.CaseClose(c.Request.Context(), a, caseID)
	if err != nil {
		log.Errorf("Could not close case. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PostContactCasesIdContinue handles POST /contact_cases/{id}/continue
func (h *server) PostContactCasesIdContinue(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactCasesIdContinue",
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
