package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// GetServiceAgentsContactInteractions handles GET /service_agents/contact_interactions
func (h *server) GetServiceAgentsContactInteractions(c *gin.Context, params openapi_server.GetServiceAgentsContactInteractionsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactInteractions",
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

	peerType := ""
	if params.PeerType != nil {
		peerType = *params.PeerType
	}

	peerTarget := ""
	if params.PeerTarget != nil {
		peerTarget = *params.PeerTarget
	}

	contactID := uuid.Nil
	if params.ContactId != nil {
		contactID = uuid.UUID(*params.ContactId)
	}

	addressID := uuid.Nil
	if params.AddressId != nil {
		addressID = uuid.UUID(*params.AddressId)
	}

	// Validate: exactly one filter mode is required.
	filterCount := 0
	if peerType != "" || peerTarget != "" {
		filterCount++
	}
	if contactID != uuid.Nil {
		filterCount++
	}
	if addressID != uuid.Nil {
		filterCount++
	}
	if filterCount > 1 {
		log.Errorf("Expected at most one filter mode, got %d.", filterCount)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_FILTER", "At most one filter is allowed: peer_type+peer_target, contact_id, or address_id."))
		return
	}
	if filterCount == 0 {
		log.Errorf("Expected at least one filter mode, got 0.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_FILTER", "At least one filter (peer_type+peer_target, contact_id, or address_id) is required."))
		return
	}

	items, nextToken, err := h.serviceHandler.ServiceAgentInteractionList(c.Request.Context(), a, pageSize, pageToken, peerType, peerTarget, contactID, addressID)
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}
