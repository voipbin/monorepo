package server

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GetServiceAgentsInteractions handles GET /service_agents/interactions
func (h *server) GetServiceAgentsInteractions(c *gin.Context, params openapi_server.GetServiceAgentsInteractionsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsInteractions",
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

	// Validate: at most one filter mode. Zero filters means "unfiltered, time-scoped" mode
	// (customer's full interaction history within the `since` window).
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

	var since time.Time
	if filterCount == 0 {
		sinceStr := "30d"
		if params.Since != nil && *params.Since != "" {
			sinceStr = *params.Since
		}
		if !strings.HasSuffix(sinceStr, "d") {
			log.Errorf("Invalid since param format: %q", sinceStr)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_SINCE", "The 'since' parameter must be in the format '<N>d' (e.g. '7d', '30d')."))
			return
		}
		n, parseErr := strconv.Atoi(strings.TrimSuffix(sinceStr, "d"))
		if parseErr != nil || n <= 0 {
			log.Errorf("Invalid since param value: %q", sinceStr)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_SINCE", "The 'since' parameter must be a positive number of days (e.g. '7d')."))
			return
		}
		if n > 180 {
			log.Errorf("since param exceeds maximum of 180d: %q", sinceStr)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_SINCE", "The 'since' parameter must be at most 180d."))
			return
		}
		since = time.Now().Add(-time.Duration(n) * 24 * time.Hour)
	}

	items, nextToken, err := h.serviceHandler.ServiceAgentInteractionList(c.Request.Context(), a, pageSize, pageToken, peerType, peerTarget, contactID, addressID, since)
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}

// GetServiceAgentsInteractionsUnresolved handles GET /service_agents/interactions/unresolved
func (h *server) GetServiceAgentsInteractionsUnresolved(c *gin.Context, params openapi_server.GetServiceAgentsInteractionsUnresolvedParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsInteractionsUnresolved",
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

	since := ""
	if params.Since != nil && *params.Since != "" {
		s := *params.Since
		if !strings.HasSuffix(s, "d") {
			log.Errorf("Invalid since param format: %q", s)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_SINCE", "The 'since' parameter must be in the format '<N>d' (e.g. '7d', '30d')."))
			return
		}
		n, parseErr := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if parseErr != nil || n <= 0 {
			log.Errorf("Invalid since param value: %q", s)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_SINCE", "The 'since' parameter must be a positive number of days (e.g. '7d')."))
			return
		}
		since = s
	}

	items, nextToken, err := h.serviceHandler.ServiceAgentInteractionListUnresolved(c.Request.Context(), a, pageSize, pageToken, since)
	if err != nil {
		log.Errorf("Could not list unresolved interactions. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}

// GetServiceAgentsInteractionsId handles GET /service_agents/interactions/{id}
func (h *server) GetServiceAgentsInteractionsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsInteractionsId",
		"request_address": c.ClientIP(),
		"id":              id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	interactionID := uuid.UUID(id)

	res, err := h.serviceHandler.ServiceAgentInteractionGet(c.Request.Context(), a, interactionID)
	if err != nil {
		log.Errorf("Could not get interaction. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PostServiceAgentsInteractionsIdResolutions handles POST /service_agents/interactions/{id}/resolutions
func (h *server) PostServiceAgentsInteractionsIdResolutions(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsInteractionsIdResolutions",
		"request_address": c.ClientIP(),
		"id":              id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	var req openapi_server.PostServiceAgentsInteractionsIdResolutionsJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	interactionID := uuid.UUID(id)
	contactID := uuid.UUID(req.ContactId)
	resolvedByID := uuid.UUID(req.ResolvedById)

	res, err := h.serviceHandler.ServiceAgentResolutionCreate(
		c.Request.Context(),
		a,
		interactionID,
		contactID,
		string(req.ResolutionType),
		string(req.ResolvedByType),
		resolvedByID,
	)
	if err != nil {
		log.Errorf("Could not create resolution. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(201, res)
}

// DeleteServiceAgentsInteractionsIdResolutionsRid handles DELETE /service_agents/interactions/{id}/resolutions/{rid}
func (h *server) DeleteServiceAgentsInteractionsIdResolutionsRid(c *gin.Context, id openapi_types.UUID, rid openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsInteractionsIdResolutionsRid",
		"request_address": c.ClientIP(),
		"id":              id,
		"rid":             rid,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	interactionID := uuid.UUID(id)
	resolutionID := uuid.UUID(rid)

	if err := h.serviceHandler.ServiceAgentResolutionDelete(c.Request.Context(), a, interactionID, resolutionID); err != nil {
		log.Errorf("Could not delete resolution. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{})
}
