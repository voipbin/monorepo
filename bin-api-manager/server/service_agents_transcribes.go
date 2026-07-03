package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// GetServiceAgentsTranscribes handles GET /service_agents/transcribes
func (h *server) GetServiceAgentsTranscribes(c *gin.Context, params openapi_server.GetServiceAgentsTranscribesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsTranscribes",
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

	tmps, err := h.serviceHandler.ServiceAgentTranscribeList(c.Request.Context(), a, pageSize, pageToken, referenceType, referenceID)
	if err != nil {
		logrus.Errorf("Could not get transcribes info. err: %v", err)
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

// PostServiceAgentsTranscribes handles POST /service_agents/transcribes
func (h *server) PostServiceAgentsTranscribes(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsTranscribes",
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

	var req openapi_server.PostServiceAgentsTranscribesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	referenceID := uuid.FromStringOrNil(req.ReferenceId)
	onEndFlowID := uuid.Nil
	if req.OnEndFlowId != nil {
		onEndFlowID = uuid.FromStringOrNil(*req.OnEndFlowId)
	}

	var provider tmtranscribe.Provider
	if req.Provider != nil {
		provider = tmtranscribe.Provider(*req.Provider)
	}

	// Default to "both" when direction is omitted or empty, and fall back to
	// "both" for a non-empty but invalid value. See server/transcribes.go's
	// PostTranscribes for the identical, previously-established reasoning.
	direction := tmtranscribe.DirectionBoth
	if req.Direction != nil && *req.Direction != "" {
		direction = tmtranscribe.Direction(*req.Direction)
		if normalized := direction.Normalize(); normalized != direction {
			log.Warnf("Invalid direction. Falling back to both. direction: %s", direction)
			direction = normalized
		}
	}

	res, err := h.serviceHandler.ServiceAgentTranscribeStart(
		c.Request.Context(),
		a,
		string(req.ReferenceType),
		referenceID,
		req.Language,
		direction,
		onEndFlowID,
		provider,
	)
	if err != nil {
		log.Errorf("Could not create a transcribe. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
