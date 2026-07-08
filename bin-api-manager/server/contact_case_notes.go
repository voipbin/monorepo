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

// GetContactCasesIdNotes handles GET /contact_cases/{id}/notes
func (h *server) GetContactCasesIdNotes(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetContactCasesIdNotes",
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

	caseID := uuid.UUID(id)

	res, err := h.serviceHandler.CaseNoteList(c.Request.Context(), a, caseID)
	if err != nil {
		log.Errorf("Could not list case notes. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, GenerateListResponse(res, ""))
}

// PostContactCasesIdNotes handles POST /contact_cases/{id}/notes
func (h *server) PostContactCasesIdNotes(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostContactCasesIdNotes",
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

	var req openapi_server.PostContactCasesIdNotesJSONRequestBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	caseID := uuid.UUID(id)

	var authorID *uuid.UUID
	if req.AuthorId != nil {
		id := uuid.UUID(*req.AuthorId)
		authorID = &id
	}

	res, err := h.serviceHandler.CaseNoteCreate(c.Request.Context(), a, caseID, string(req.AuthorType), authorID, req.Text)
	if err != nil {
		log.Errorf("Could not create case note. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// DeleteContactCasesIdNotesNoteId handles DELETE /contact_cases/{id}/notes/{note_id}
func (h *server) DeleteContactCasesIdNotesNoteId(c *gin.Context, id openapi_types.UUID, noteId openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteContactCasesIdNotesNoteId",
		"request_address": c.ClientIP(),
		"id":              id,
		"note_id":         noteId,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("customer_id", a.CustomerID)

	caseID := uuid.UUID(id)
	noteID := uuid.UUID(noteId)

	if err := h.serviceHandler.CaseNoteDelete(c.Request.Context(), a, caseID, noteID); err != nil {
		log.Errorf("Could not delete case note. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, gin.H{})
}
