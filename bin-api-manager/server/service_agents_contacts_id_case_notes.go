package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GetServiceAgentsContactsIdCaseNotes handles GET /service_agents/contacts/{id}/case_notes
func (h *server) GetServiceAgentsContactsIdCaseNotes(c *gin.Context, id openapi_types.UUID, params openapi_server.GetServiceAgentsContactsIdCaseNotesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactsIdCaseNotes",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid contact ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

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

	tmps, nextToken, err := h.serviceHandler.ServiceAgentContactCaseNoteList(c.Request.Context(), a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not list the case notes of the contact. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(tmps, nextToken))
}
