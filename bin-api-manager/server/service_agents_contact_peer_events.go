package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// GetServiceAgentsContactPeerEvents handles GET /service_agents/contact_peer_events.
// Mirrors GetContactPeerEvents but calls the ServiceAgentPeerEventList
// servicehandler method (PermissionAll gate, a.CustomerID directly).
func (h *server) GetServiceAgentsContactPeerEvents(c *gin.Context, params openapi_server.GetServiceAgentsContactPeerEventsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactPeerEvents",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize == 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = string(*params.PageToken)
	}

	contactID := uuid.Nil
	if params.ContactId != nil {
		contactID = uuid.UUID(*params.ContactId)
	}

	peerType := ""
	if params.PeerType != nil {
		peerType = *params.PeerType
	}

	peerTarget := ""
	if params.PeerTarget != nil {
		peerTarget = *params.PeerTarget
	}

	filterCount := 0
	if contactID != uuid.Nil {
		filterCount++
	}
	if peerType != "" || peerTarget != "" {
		filterCount++
	}
	if filterCount != 1 {
		log.Errorf("Expected exactly one filter mode, got %d.", filterCount)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_FILTER", "Exactly one filter is required: contact_id, or peer_type+peer_target."))
		return
	}

	items, nextToken, err := h.serviceHandler.ServiceAgentPeerEventList(c.Request.Context(), a, contactID, peerType, peerTarget, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not list peer events. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}
