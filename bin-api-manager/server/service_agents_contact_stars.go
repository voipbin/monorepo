package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// PutServiceAgentsContactsIdStar handles PUT /service_agents/contacts/{id}/star
func (h *server) PutServiceAgentsContactsIdStar(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsContactsIdStar",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{"agent": a})

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid contact ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentContactStarSet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not star the contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteServiceAgentsContactsIdStar handles DELETE /service_agents/contacts/{id}/star
func (h *server) DeleteServiceAgentsContactsIdStar(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsContactsIdStar",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{"agent": a})

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid contact ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentContactStarDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not remove the star. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// GetServiceAgentsContactStars handles GET /service_agents/contact_stars
func (h *server) GetServiceAgentsContactStars(c *gin.Context, params openapi_server.GetServiceAgentsContactStarsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactStars",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{"agent": a})

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, nextToken, err := h.serviceHandler.ServiceAgentContactStarList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not list the stars. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(tmps, nextToken))
}
