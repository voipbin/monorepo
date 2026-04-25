package server

import (
	"net/http"

	"monorepo/bin-api-manager/gens/openapi_server"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cfconference "monorepo/bin-conference-manager/models/conference"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

func (h *server) GetConferences(c *gin.Context, params openapi_server.GetConferencesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConferences",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

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

	tmps, err := h.serviceHandler.ConferenceList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get conferences. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) PostConferences(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferences",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	var req openapi_server.PostConferencesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	id := uuid.Nil
	if req.Id != nil {
		id = uuid.FromStringOrNil(*req.Id)
	}
	preFlowID := uuid.FromStringOrNil(req.PreFlowId)
	postFlowID := uuid.FromStringOrNil(req.PostFlowId)

	res, err := h.serviceHandler.ConferenceCreate(
		c.Request.Context(),
		a,
		id,
		cfconference.Type(req.Type),
		req.Name,
		req.Detail,
		req.Data,
		req.Timeout,
		preFlowID,
		postFlowID,
	)
	if err != nil {
		log.Errorf("Could not create the conference. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetConferencesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConferencesId",
		"request_address": c.ClientIP,
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ConferenceGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get the conference. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutConferencesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "conferencesIDPUT",
		"request_address": c.ClientIP,
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutConferencesIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	preFlowID := uuid.FromStringOrNil(req.PreFlowId)
	postFlowID := uuid.FromStringOrNil(req.PostFlowId)

	res, err := h.serviceHandler.ConferenceUpdate(c.Request.Context(), a, target, req.Name, req.Detail, req.Data, req.Timeout, preFlowID, postFlowID)
	if err != nil {
		log.Errorf("Could not update the conference. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteConferencesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteConferencesId",
		"request_address": c.ClientIP,
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ConferenceDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete the conference. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostConferencesIdRecordingStart(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdRecordingStart",
		"request_address": c.ClientIP,
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PostConferencesIdRecordingStartJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}
	onEndFlowID := uuid.FromStringOrNil(req.OnEndFlowId)

	res, err := h.serviceHandler.ConferenceRecordingStart(c.Request.Context(), a, target, cmrecording.Format(req.Format), req.Duration, onEndFlowID)
	if err != nil {
		log.Errorf("Could not start the conference recording. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostConferencesIdRecordingStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdRecordingStop",
		"request_address": c.ClientIP,
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ConferenceRecordingStop(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not stop the conference recording. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostConferencesIdTranscribeStart(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdTranscribeStart",
		"request_address": c.ClientIP,
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PostConferencesIdTranscribeStartJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ConferenceTranscribeStart(c.Request.Context(), a, target, req.Language)
	if err != nil {
		log.Errorf("Could not start the conference recording. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostConferencesIdTranscribeStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdTranscribeStop",
		"request_address": c.ClientIP,
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ConferenceTranscribeStop(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not stop the conference transcribe. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetConferencesIdMediaStream(c *gin.Context, id string, params openapi_server.GetConferencesIdMediaStreamParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetConferencesIdMediaStream",
		"request_address": c.ClientIP,
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	if errMedia := h.serviceHandler.ConferenceMediaStreamStart(c.Request.Context(), a, target, params.Encapsulation, c.Writer, c.Request); errMedia != nil {
		log.Errorf("Could not start the conference media streaming. err: %v", errMedia)
		abortWithServiceError(c, errMedia)
		return
	}

	c.Status(http.StatusOK)
}

func (h *server) PostConferencesIdDirectHashRegenerate(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConferencesIdDirectHashRegenerate",
		"request_address": c.ClientIP(),
		"conference_id":   id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	// Convert openapi_types.UUID to uuid.UUID
	conferenceID, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid conference ID format. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ConferenceDirectHashRegenerate(c.Request.Context(), a, conferenceID)
	if err != nil {
		log.Errorf("Could not regenerate conference direct hash. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
