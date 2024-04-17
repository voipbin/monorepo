package recordingfiles

import (
	"net/http"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// recordingfilesIDGET handles GET /recordingfiles/<id> request.
// It returns recording file.
//
//	@Summary		Download the recording file
//	@Description	Download the recording file
//	@Produce		json
//	@Param			id	query	string	true	"The recordingfile's id."
//	@Success		200	"recording file"
//	@Router			/v1.0/recordingfiles/{id} [get]
func recordingfilesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "recordingfilesIDGET",
		"request_address": c.ClientIP,
	})

	tmpAgent, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmpAgent.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("recordingfile_id", id)
	log.Debug("Executing recordingfilesIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	downloadURI, err := serviceHandler.RecordingfileGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a recordingfile. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, downloadURI)
}
