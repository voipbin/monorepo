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

// PostContactCasesIdResolutions handles POST /contact_cases/{id}/resolutions
// (VOIP-1252): attaches a case to a contact.
func (h *server) PostContactCasesIdResolutions(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactCasesIdResolutions",
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

	var req openapi_server.PostContactCasesIdResolutionsJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	caseID := uuid.UUID(id)
	contactID := uuid.UUID(req.ContactId)

	res, err := h.serviceHandler.CaseResolutionCreate(c.Request.Context(), a, caseID, contactID, string(req.ResolutionType))
	if err != nil {
		log.Errorf("Could not create case resolution. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteContactCasesIdResolutionsResolutionId handles
// DELETE /contact_cases/{id}/resolutions/{resolution_id} (VOIP-1252): undoes
// a case-level Contact attribution.
func (h *server) DeleteContactCasesIdResolutionsResolutionId(c *gin.Context, id openapi_types.UUID, resolutionId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactCasesIdResolutionsResolutionId",
		"request_address": c.ClientIP(),
		"id":              id,
		"resolution_id":   resolutionId,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	caseID := uuid.UUID(id)
	resolutionID := uuid.UUID(resolutionId)

	if err := h.serviceHandler.CaseResolutionDelete(c.Request.Context(), a, caseID, resolutionID); err != nil {
		log.Errorf("Could not delete case resolution. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{})
}
