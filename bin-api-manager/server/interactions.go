package server

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GetInteractions handles GET /interactions
func (h *server) GetInteractions(c *gin.Context, params openapi_server.GetInteractionsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetInteractions",
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

	res, err := h.serviceHandler.InteractionList(c.Request.Context(), a, pageSize, pageToken, peerType, peerTarget, contactID, addressID)
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// GetInteractionsUnresolved handles GET /interactions/unresolved
func (h *server) GetInteractionsUnresolved(c *gin.Context, params openapi_server.GetInteractionsUnresolvedParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetInteractionsUnresolved",
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

	// Validate and pass since param directly in "Nd" format.
	// Empty → "" (backend applies default 30d). Max 180d enforced here and in listenhandler.
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
		if n > 180 {
			log.Errorf("since param exceeds maximum: %q", s)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_SINCE", "The 'since' parameter must not exceed 180d."))
			return
		}
		since = s
	}

	res, err := h.serviceHandler.InteractionListUnresolved(c.Request.Context(), a, pageSize, pageToken, since)
	if err != nil {
		log.Errorf("Could not list unresolved interactions. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// GetInteractionsId handles GET /interactions/{id}
func (h *server) GetInteractionsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetInteractionsId",
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

	res, err := h.serviceHandler.InteractionGet(c.Request.Context(), a, interactionID)
	if err != nil {
		log.Errorf("Could not get interaction. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PostInteractionsIdResolutions handles POST /interactions/{id}/resolutions
func (h *server) PostInteractionsIdResolutions(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostInteractionsIdResolutions",
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

	var req openapi_server.PostInteractionsIdResolutionsJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	interactionID := uuid.UUID(id)
	contactID := uuid.UUID(req.ContactId)
	resolvedByID := uuid.UUID(req.ResolvedById)

	res, err := h.serviceHandler.ResolutionCreate(
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

// DeleteInteractionsIdResolutionsRid handles DELETE /interactions/{id}/resolutions/{rid}
func (h *server) DeleteInteractionsIdResolutionsRid(c *gin.Context, id openapi_types.UUID, rid openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteInteractionsIdResolutionsRid",
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

	if err := h.serviceHandler.ResolutionDelete(c.Request.Context(), a, interactionID, resolutionID); err != nil {
		log.Errorf("Could not delete resolution. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{})
}
