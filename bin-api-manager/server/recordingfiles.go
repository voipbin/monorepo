package server

import (
	"net/http"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) GetRecordingfilesId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "GetRecordingfilesId",
		"request_address":  c.ClientIP,
		"recordingfile_id": id,
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

	downloadURI, err := h.serviceHandler.RecordingfileGet(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not get a recordingfile. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, downloadURI)
}
