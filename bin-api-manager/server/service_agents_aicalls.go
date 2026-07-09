package server

import (
	amaicall "monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// GetServiceAgentsAicalls handles GET /service_agents/aicalls
func (h *server) GetServiceAgentsAicalls(c *gin.Context, params openapi_server.GetServiceAgentsAicallsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsAicalls",
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

	referenceType := ""
	if params.ReferenceType != nil {
		referenceType = string(*params.ReferenceType)
	}

	referenceID := uuid.Nil
	if params.ReferenceId != nil {
		referenceID = uuid.UUID(*params.ReferenceId)
	}

	// reference_type and reference_id are documented as a pair (see the
	// OpenAPI description); reject a partial filter explicitly instead of
	// silently applying only one half of the intended filter.
	if (referenceType == "") != (referenceID == uuid.Nil) {
		log.Errorf("reference_type and reference_id must be supplied together. reference_type: %s, reference_id: %s", referenceType, referenceID)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_REFERENCE_FILTER", "reference_type and reference_id must be supplied together."))
		return
	}

	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}

	tmps, err := h.serviceHandler.ServiceAgentAIcallList(c.Request.Context(), a, pageSize, pageToken, referenceType, referenceID, status)
	if err != nil {
		logrus.Errorf("Could not get aicalls info. err: %v", err)
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

// PostServiceAgentsAicalls handles POST /service_agents/aicalls
func (h *server) PostServiceAgentsAicalls(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsAicalls",
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

	var req openapi_server.PostServiceAgentsAicallsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	assistanceID := uuid.UUID(req.AssistanceId)
	referenceID := uuid.UUID(req.ReferenceId)

	res, err := h.serviceHandler.ServiceAgentAIcallCreate(
		c.Request.Context(),
		a,
		amaicall.AssistanceType(req.AssistanceType),
		assistanceID,
		amaicall.ReferenceType(req.ReferenceType),
		referenceID,
	)
	if err != nil {
		log.Errorf("Could not create aicall. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
