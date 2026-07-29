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

// GetServiceAgentsContactCasesIdNotes handles GET /service_agents/contact_cases/{id}/notes
func (h *server) GetServiceAgentsContactCasesIdNotes(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetServiceAgentsContactCasesIdNotes",
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
		log.Errorf("Invalid case ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentCaseNoteList(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not list case notes. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(res, ""))
}

// PostServiceAgentsContactCasesIdNotes handles POST /service_agents/contact_cases/{id}/notes
func (h *server) PostServiceAgentsContactCasesIdNotes(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsContactCasesIdNotes",
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
		log.Errorf("Invalid case ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	var req openapi_server.PostServiceAgentsContactCasesIdNotesJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentCaseNoteCreate(c.Request.Context(), a, target, req.Text)
	if err != nil {
		log.Errorf("Could not create case note. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteServiceAgentsContactCasesIdNotesNoteId handles DELETE /service_agents/contact_cases/{id}/notes/{note_id}
func (h *server) DeleteServiceAgentsContactCasesIdNotesNoteId(c *gin.Context, id openapi_types.UUID, noteId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteServiceAgentsContactCasesIdNotesNoteId",
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
		log.Errorf("Invalid case ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	noteID, err := uuid.FromString(noteId.String())
	if err != nil {
		log.Errorf("Invalid note ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided note_id is not a valid UUID.").Wrap(err))
		return
	}

	if err := h.serviceHandler.ServiceAgentCaseNoteDelete(c.Request.Context(), a, target, noteID); err != nil {
		log.Errorf("Could not delete case note. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{})
}
