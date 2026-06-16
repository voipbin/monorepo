package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostTranscribes(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostTranscribes",
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

	var req openapi_server.PostTranscribesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	referenceID := uuid.FromStringOrNil(req.ReferenceId)
	onEndFlowID := uuid.Nil
	if req.OnEndFlowId != nil {
		onEndFlowID = uuid.FromStringOrNil(*req.OnEndFlowId)
	}

	var provider tmtranscribe.Provider
	if req.Provider != nil {
		provider = tmtranscribe.Provider(*req.Provider)
	}

	// Default to "both" when direction is omitted. direction is optional in the
	// spec; before this handler honored it, the value was always DirectionBoth.
	// Preserve that so callers that omit direction keep capturing both legs
	// instead of a broken empty-direction stream.
	direction := tmtranscribe.DirectionBoth
	if req.Direction != nil {
		direction = tmtranscribe.Direction(*req.Direction)
	}

	res, err := h.serviceHandler.TranscribeStart(
		c.Request.Context(),
		a,
		string(req.ReferenceType),
		referenceID,
		req.Language,
		direction,
		onEndFlowID,
		provider,
	)
	if err != nil {
		log.Errorf("Could not create a transcribe. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetTranscribes(c *gin.Context, params openapi_server.GetTranscribesParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTranscribes",
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

	tmps, err := h.serviceHandler.TranscribeList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get transcribes info. err: %v", err)
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

func (h *server) GetTranscribesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTranscribesId",
		"request_address": c.ClientIP,
		"transcribe_id":   id,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.TranscribeGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a transcribe. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteTranscribesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteTranscribesId",
		"request_address": c.ClientIP,
		"transcribe_id":   id,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.TranscribeDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not delete the transcribe. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostTranscribesIdStop(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostTranscribesIdStop",
		"request_address": c.ClientIP,
		"transcribe_id":   id,
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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.TranscribeStop(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not stop the transcribe. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
