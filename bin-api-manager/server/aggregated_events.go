package server

import (
	"net/http"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetAggregatedEvents(c *gin.Context, params openapi_server.GetAggregatedEventsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetAggregatedEvents",
		"request_address": c.ClientIP(),
	})

	// Get agent from context
	tmp, exists := c.Get("agent")
	if !exists {
		log.Error("Could not find agent info")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "authentication required"})
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithField("customer_id", a.CustomerID)

	// Parse query params
	var activeflowID, callID uuid.UUID
	if params.ActiveflowId != nil {
		parsed, err := uuid.FromString(params.ActiveflowId.String())
		if err != nil {
			log.Infof("Invalid activeflow_id: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid activeflow_id"})
			return
		}
		activeflowID = parsed
	}
	if params.CallId != nil {
		parsed, err := uuid.FromString(params.CallId.String())
		if err != nil {
			log.Infof("Invalid call_id: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid call_id"})
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
	events, nextPageToken, err := h.serviceHandler.AggregatedEventList(c.Request.Context(), &a, activeflowID, callID, pageSize, pageToken)
	if err != nil {
		log.Infof("Failed to get aggregated events: %v", err)
		if err.Error() == "user has no permission" || err.Error() == "not found" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "resource not found"})
			return
		}
		if err.Error() == "either activeflow_id or call_id is required" || err.Error() == "only one of activeflow_id or call_id is allowed" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "internal error"})
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
