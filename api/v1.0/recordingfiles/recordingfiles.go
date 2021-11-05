package recordingfiles

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// recordingfilesIDGET handles GET /recordingfiles/<id> request.
// It returns recording file.
// @Summary Download the recording file
// @Description Download the recording file
// @Produce json
// @Success 200
// @Router /v1.0/recordingfiles/{id} [get]
func recordingfilesIDGET(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}

	// get user
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})
	log.Debug("Executing recordingfilesIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	url, err := serviceHandler.RecordingfileGet(&u, id)
	if err != nil {
		log.Errorf("Could not get a recordingfile. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}
