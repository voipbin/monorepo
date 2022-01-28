package transcribes

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// transcribesPOST handles POST /transcribes request.
// It creates a transcribe of the recording and returns the result.
// @Summary Create a transcribe
// @Description transcribe a recording
// @Produce json
// @Param transcribe body request.BodyTranscribesPOST true "Creating transcribe info."
// @Success 200 {object} transcribe.Transcribe
// @Router /v1.0/transcribes [post]
func transcribesPOST(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "agentsGET",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("customer")
	if !exists {
		log.Errorf("Could not find customer info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(cscustomer.Customer)
	log = log.WithFields(
		logrus.Fields{
			"customer_id": u.ID,
			"username":    u.Username,
		},
	)

	var req request.BodyTranscribesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing transcribesPOST.")

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create a transcribe
	res, err := serviceHandler.TranscribeCreate(&u, req.RecordingID, req.Language)
	if err != nil {
		log.Errorf("Could not create a transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
