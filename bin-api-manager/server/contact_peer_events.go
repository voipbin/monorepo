package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// GetContactPeerEvents handles GET /contact_peer_events.
// Unlike GET /contact_interactions, this returns raw peer_events rows with
// NO identity resolution and NO CRM eligibility filtering — see
// docs/plans/2026-07-24-peer-events-read-api-design.md for the full contract.
func (h *server) GetContactPeerEvents(c *gin.Context, params openapi_server.GetContactPeerEventsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactPeerEvents",
		"request_address": c.ClientIP(),
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	// page_size default/max: 100, clamped to [1,100] — same clamp as
	// GetContactInteractions (server/contact_interactions.go:33-39), the
	// bin-api-manager HTTP-layer ceiling, distinct from peereventhandler's
	// internal 100/1000 clamp.
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

	// Validate: exactly one filter mode must be provided.
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

	items, nextToken, err := h.serviceHandler.PeerEventList(c.Request.Context(), a, contactID, peerType, peerTarget, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not list peer events. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}
