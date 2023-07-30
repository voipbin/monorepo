package recordingfiles

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// recordingfilesIDGET handles GET /recordingfiles/<id> request.
// It returns recording file.
// @Summary     Download the recording file
// @Description Download the recording file
// @Produce     json
// @Param       id  query string true "The recordingfile's id."
// @Success     200 "recording file"
// @Router      /v1.0/recordingfiles/{id} [get]
func recordingfilesIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "recordingfilesIDGET",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(logrus.Fields{
		"customer_id":    u.ID,
		"username":       u.Username,
		"permission_ids": u.PermissionIDs,
	})

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("recordingfile_id", id)
	log.Debug("Executing recordingfilesIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	url, err := serviceHandler.RecordingfileGet(c.Request.Context(), &u, id)
	if err != nil {
		log.Errorf("Could not get a recordingfile. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}
