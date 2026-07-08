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

// GetContactInteractionsUnresolved handles GET /contact_interactions/unresolved
func (h *server) GetContactInteractionsUnresolved(c *gin.Context, params openapi_server.GetContactInteractionsUnresolvedParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactInteractionsUnresolved",
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
	// Empty → "" (backend applies default 30d). Format and positive-int validated here;
	// upper-bound (180d) is enforced by the contact-manager listenhandler.
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

	items, nextToken, err := h.serviceHandler.InteractionListUnresolved(c.Request.Context(), a, pageSize, pageToken, since)
	if err != nil {
		log.Errorf("Could not list unresolved interactions. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(items, nextToken))
}

// GetContactInteractionsId handles GET /contact_interactions/{id}
func (h *server) GetContactInteractionsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactInteractionsId",
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

// PostContactInteractionsIdResolutions handles POST /contact_interactions/{id}/resolutions
func (h *server) PostContactInteractionsIdResolutions(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactInteractionsIdResolutions",
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

	var req openapi_server.PostContactInteractionsIdResolutionsJSONRequestBody
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

// DeleteContactInteractionsIdResolutionsRid handles DELETE /contact_interactions/{id}/resolutions/{rid}
func (h *server) DeleteContactInteractionsIdResolutionsRid(c *gin.Context, id openapi_types.UUID, rid openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactInteractionsIdResolutionsRid",
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
