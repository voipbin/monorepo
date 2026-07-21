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

// GetServiceAgentsContactCases handles GET /service_agents/contact_cases
func (h *server) GetServiceAgentsContactCases(c *gin.Context, params openapi_server.GetServiceAgentsContactCasesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactCases",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
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

	tmps, nextToken, err := h.serviceHandler.ServiceAgentCaseList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get cases info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

// GetServiceAgentsContactCasesId handles GET /service_agents/contact_cases/{id}
func (h *server) GetServiceAgentsContactCasesId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactCasesId",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid case ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentCaseGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a case. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PostServiceAgentsContactCasesIdClose handles POST /service_agents/contact_cases/{id}/close
func (h *server) PostServiceAgentsContactCasesIdClose(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsContactCasesIdClose",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid case ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentCaseClose(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not close the case. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PostServiceAgentsContactCasesIdAssign handles POST /service_agents/contact_cases/{id}/assign
func (h *server) PostServiceAgentsContactCasesIdAssign(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsContactCasesIdAssign",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid case ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	var req openapi_server.PostServiceAgentsContactCasesIdAssignJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	ownerID, err := uuid.FromString(req.OwnerId.String())
	if err != nil {
		log.Errorf("Invalid owner ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_OWNER_ID", "The provided owner_id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentCaseAssign(c.Request.Context(), a, target, ownerID)
	if err != nil {
		log.Errorf("Could not assign the case. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
