package server

import (
	"net/http"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) GetTimelinesResourceTypeResourceIdEvents(c *gin.Context, resourceType openapi_server.GetTimelinesResourceTypeResourceIdEventsParamsResourceType, resourceId openapi_types.UUID, params openapi_server.GetTimelinesResourceTypeResourceIdEventsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTimelinesResourceTypeResourceIdEvents",
		"request_address": c.ClientIP(),
		"resource_type":   resourceType,
		"resource_id":     resourceId,
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

	// Validate resource_type
	validTypes := map[openapi_server.GetTimelinesResourceTypeResourceIdEventsParamsResourceType]bool{
		openapi_server.Calls:       true,
		openapi_server.Conferences: true,
		openapi_server.Flows:       true,
		openapi_server.Activeflows: true,
	}
	if !validTypes[resourceType] {
		log.Info("Invalid resource type")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid resource type"})
		return
	}

	// Convert openapi_types.UUID to uuid.UUID
	resourceUUID, err := uuid.FromString(resourceId.String())
	if err != nil {
		log.Infof("Invalid resource id: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid resource id"})
		return
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
	events, nextPageToken, err := h.serviceHandler.TimelineEventList(c.Request.Context(), &a, string(resourceType), resourceUUID, pageSize, pageToken)
	if err != nil {
		log.Infof("Failed to get timeline events: %v", err)
		if err.Error() == "user has no permission" || err.Error() == "not found" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "resource not found"})
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
