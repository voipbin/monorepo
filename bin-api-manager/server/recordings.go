package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// recordingsGET handles GET /recordings request.
// It returns list of calls of the given customer.
//
//	@Summary		List recordings
//	@Description	get recordings of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyRecordingsGET
//	@Router			/v1.0/recordings [get]
func (h *server) GetRecordings(c *gin.Context, params openapi_server.GetRecordingsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRecordings",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

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

	tmps, err := h.serviceHandler.RecordingList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a recordings. err: %v", err)
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

func (h *server) GetRecordingsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRecordingsId",
		"request_address": c.ClientIP,
		"recording_id":    id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, err := h.serviceHandler.RecordingGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a recording info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteRecordingsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteRecordingsId",
		"request_address": c.ClientIP,
		"recording_id":    id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	res, err := h.serviceHandler.RecordingDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a recording info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// GetRecordingsIdTranscribes handles GET /recordings/{id}/transcribes request.
// It returns the transcribes tied to the recording regardless of the transcribe
// owner (including transcribes created internally for AI summaries), gated by
// ownership of the recording itself.
func (h *server) GetRecordingsIdTranscribes(c *gin.Context, id string, params openapi_server.GetRecordingsIdTranscribesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRecordingsIdTranscribes",
		"request_address": c.ClientIP,
		"recording_id":    id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
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

	res, err := h.serviceHandler.RecordingTranscribeList(c.Request.Context(), a, target, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get transcribes for the recording. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(res) > 0 && res[len(res)-1].TMCreate != nil {
		nextToken = res[len(res)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
	}

	c.JSON(200, GenerateListResponse(res, nextToken))
}

// GetRecordingsIdTranscripts handles GET /recordings/{id}/transcripts request.
// It returns the transcript lines for a transcribe tied to the recording,
// gated by ownership of the recording and by the transcribe belonging to it.
func (h *server) GetRecordingsIdTranscripts(c *gin.Context, id string, params openapi_server.GetRecordingsIdTranscriptsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRecordingsIdTranscripts",
		"request_address": c.ClientIP,
		"recording_id":    id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(
			commonoutline.ServiceNameAPIManager,
			"AUTHENTICATION_REQUIRED",
			"Authentication is required.",
		))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_ID",
			"The provided id is not a valid UUID.",
		))
		return
	}

	transcribeID := uuid.FromStringOrNil(params.TranscribeId)
	if transcribeID == uuid.Nil {
		log.Error("Could not parse the transcribe_id.")
		abortWithError(c, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_TRANSCRIBE_ID",
			"The provided transcribe_id is not a valid UUID.",
		))
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

	res, err := h.serviceHandler.RecordingTranscriptList(c.Request.Context(), a, target, transcribeID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get transcripts for the recording. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(res) > 0 && res[len(res)-1].TMCreate != nil {
		nextToken = res[len(res)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
	}

	c.JSON(200, GenerateListResponse(res, nextToken))
}
