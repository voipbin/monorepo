package server

import (
	"errors"
	"io"
	"net/http"

	"monorepo/bin-api-manager/gens/openapi_server"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// PostServiceAgentsCallsIdRecordingStart starts the recording of a call the calling agent owns.
// The body is optional: every field has a server-side default.
func (h *server) PostServiceAgentsCallsIdRecordingStart(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsCallsIdRecordingStart",
		"request_address": c.ClientIP(),
		"call_id":         id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid call ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	var req openapi_server.PostServiceAgentsCallsIdRecordingStartJSONBody
	if c.Request.Body != nil && c.Request.Body != http.NoBody {
		if errBind := c.ShouldBindJSON(&req); errBind != nil && !errors.Is(errBind, io.EOF) {
			log.Errorf("Could not parse the request parameter. err: %v", errBind)
			abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON."))
			return
		}
	}

	var format *cmrecording.Format
	if req.Format != nil {
		f := cmrecording.Format(*req.Format)
		format = &f
	}

	res, err := h.serviceHandler.ServiceAgentCallRecordingStart(c.Request.Context(), a, target, format, req.EndOfSilence, req.EndOfKey, req.Duration)
	if err != nil {
		log.Errorf("Could not start the recording on the call. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// PostServiceAgentsCallsIdRecordingStop stops the recording of a call the calling agent owns.
func (h *server) PostServiceAgentsCallsIdRecordingStop(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsCallsIdRecordingStop",
		"request_address": c.ClientIP(),
		"call_id":         id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid call ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentCallRecordingStop(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not stop the recording on the call. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
