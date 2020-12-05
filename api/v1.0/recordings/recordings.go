package recordings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/api"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/servicehandler"
)

// recordingsIDGET handles GET /recordings/<id> request.
// It gets a list of flows with the given info.
// @Summary Gets a list of flows.
// @Description Gets a list of flows
// @Produce json
// @Success 200 {array} flow.Flow
// @Router /v1.0/flows [get]
func recordingsIDGET(c *gin.Context) {

	// get id
	id := c.Params.ByName("id")

	tmp, exists := c.Get("user")
	if exists != true {
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
	log.Debug("Executing recordingsIDGET.")

	serviceHandler := c.MustGet(api.OBJServiceHandler).(servicehandler.ServiceHandler)
	url, err := serviceHandler.RecordingGet(&u, id)
	if err != nil {
		log.Errorf("Could not get a flow. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}
