package server

import (
	"net/http"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetAggregatedEvents(c *gin.Context, params openapi_server.GetAggregatedEventsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAggregatedEvents",
		"request_address": c.ClientIP(),
	})

	// Get auth identity from context
	a, ok := getAuthIdentity(c)
	if !ok {
		log.Error("Could not find auth identity")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	// Parse query params
	var activeflowID, callID uuid.UUID
	if params.ActiveflowId != nil {
		parsed, err := uuid.FromString(params.ActiveflowId.String())
		if err != nil {
			log.Infof("Invalid activeflow_id: %v", err)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ACTIVEFLOW_ID", "The provided activeflow_id is not a valid UUID.").Wrap(err))
			return
		}
		activeflowID = parsed
	}
	if params.CallId != nil {
		parsed, err := uuid.FromString(params.CallId.String())
		if err != nil {
			log.Infof("Invalid call_id: %v", err)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_CALL_ID", "The provided call_id is not a valid UUID.").Wrap(err))
			return
		}
		callID = parsed
	}

	// Parse pagination params
	pageSize := 100
	if params.PageSize != nil {
		pageSize = *params.PageSize
		if pageSize <= 0 || pageSize > 1000 {
			pageSize = 100
		}
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	// Call servicehandler
	events, nextPageToken, err := h.serviceHandler.AggregatedEventList(c.Request.Context(), a, activeflowID, callID, pageSize, pageToken)
	if err != nil {
		log.Infof("Failed to get aggregated events: %v", err)
		abortWithServiceError(c, err)
		return
	}

	// Build response
	res := struct {
		Result        interface{} `json:"result"`
		NextPageToken string      `json:"next_page_token,omitempty"`
	}{
		Result:        events,
		NextPageToken: nextPageToken,
	}

	c.JSON(http.StatusOK, res)
}
