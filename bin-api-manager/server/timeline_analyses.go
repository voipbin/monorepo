package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tmanalysis "monorepo/bin-timeline-manager/models/analysis"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// PostTimelineAnalyses handles POST /timeline-analyses — trigger an AI analysis
// of an ended activeflow. The customer_id is injected from the authenticated
// token (never from the client). Requires CustomerAdmin+ (enforced in the
// servicehandler).
func (h *server) PostTimelineAnalyses(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostTimelineAnalyses",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{"auth": a})

	var req openapi_server.PostTimelineAnalysesJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	activeflowID := uuid.FromStringOrNil(req.ActiveflowId)

	reanalyze := false
	if req.Reanalyze != nil {
		reanalyze = *req.Reanalyze
	}

	res, err := h.serviceHandler.TimelineAnalysisCreate(c.Request.Context(), a, activeflowID, reanalyze)
	if err != nil {
		log.Errorf("Could not create timeline analysis. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// GetTimelineAnalyses handles GET /timeline-analyses — list the authenticated
// customer's analyses.
func (h *server) GetTimelineAnalyses(c *gin.Context, params openapi_server.GetTimelineAnalysesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTimelineAnalyses",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{"auth": a})

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

	activeflowID := uuid.Nil
	if params.ActiveflowId != nil {
		activeflowID = uuid.UUID(*params.ActiveflowId)
	}

	status := tmanalysis.Status("")
	if params.Status != nil {
		status = tmanalysis.Status(*params.Status)
	}

	tmps, err := h.serviceHandler.TimelineAnalysisGetsByCustomerID(c.Request.Context(), a, pageSize, pageToken, activeflowID, status)
	if err != nil {
		log.Errorf("Could not get timeline analysis list. err: %v", err)
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

// GetTimelineAnalysesId handles GET /timeline-analyses/{id} — get one analysis
// (ownership-checked; masked not-found on mismatch).
func (h *server) GetTimelineAnalysesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTimelineAnalysesId",
		"request_address": c.ClientIP,
		"analysis_id":     id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{"auth": a})

	target := uuid.FromStringOrNil(id)

	res, err := h.serviceHandler.TimelineAnalysisGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get timeline analysis. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteTimelineAnalysesId handles DELETE /timeline-analyses/{id} — soft-delete
// (ownership-checked). Requires CustomerAdmin+ (enforced in the servicehandler).
func (h *server) DeleteTimelineAnalysesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteTimelineAnalysesId",
		"request_address": c.ClientIP,
		"analysis_id":     id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{"auth": a})

	target := uuid.FromStringOrNil(id)

	res, err := h.serviceHandler.TimelineAnalysisDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete timeline analysis. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
