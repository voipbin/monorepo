package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"net/http"

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

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	downloadURI, err := h.serviceHandler.RecordingfileGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a recordingfile. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, downloadURI)
}
