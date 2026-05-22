package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// GetAisIdPromptHistories handles GET /ais/{id}/prompt_histories
func (h *server) GetAisIdPromptHistories(c *gin.Context, id string, params openapi_server.GetAisIdPromptHistoriesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAisIdPromptHistories",
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

	aiID := uuid.FromStringOrNil(id)
	if aiID == uuid.Nil {
		log.Error("Could not parse the ai id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided ai id is not a valid UUID."))
		return
	}

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

	tmps, err := h.serviceHandler.AIPromptHistoryGetsByAIID(c.Request.Context(), a, aiID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get ai prompt history list. err: %v", err)
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

// GetAisIdPromptHistoriesHistoryId handles GET /ais/{id}/prompt_histories/{history_id}
func (h *server) GetAisIdPromptHistoriesHistoryId(c *gin.Context, id string, historyId string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAisIdPromptHistoriesHistoryId",
		"request_address": c.ClientIP,
		"ai_id":           id,
		"history_id":      historyId,
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

	aiID := uuid.FromStringOrNil(id)
	if aiID == uuid.Nil {
		log.Error("Could not parse the ai id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided ai id is not a valid UUID."))
		return
	}

	historyID := uuid.FromStringOrNil(historyId)
	if historyID == uuid.Nil {
		log.Error("Could not parse the history id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_HISTORY_ID", "The provided history id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.AIPromptHistoryGet(c.Request.Context(), a, aiID, historyID)
	if err != nil {
		log.Errorf("Could not get ai prompt history. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
