package server

import (
	"net/http"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

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
	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
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
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_RESOURCE_TYPE", "The provided resource_type is not supported."))
		return
	}

	// Convert openapi_types.UUID to uuid.UUID
	resourceUUID, err := uuid.FromString(resourceId.String())
	if err != nil {
		log.Infof("Invalid resource id: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
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
	events, nextPageToken, err := h.serviceHandler.TimelineEventList(c.Request.Context(), a, string(resourceType), resourceUUID, pageSize, pageToken)
	if err != nil {
		log.Infof("Failed to get timeline events: %v", err)
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
