package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// GetContactInteractions handles GET /contact_interactions
func (h *server) GetContactInteractions(c *gin.Context, params openapi_server.GetContactInteractionsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactInteractions",
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

	// Validate: exactly one filter mode must be provided.
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
	if filterCount != 1 {
		log.Errorf("Expected exactly one filter mode, got %d.", filterCount)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_FILTER", "Exactly one filter is required: peer_type+peer_target, contact_id, or address_id."))
		return
	}

	items, nextToken, err := h.serviceHandler.InteractionList(c.Request.Context(), a, pageSize, pageToken, peerType, peerTarget, contactID, addressID)
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}
