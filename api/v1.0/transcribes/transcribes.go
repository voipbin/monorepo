package transcribes

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// transcribesPOST handles POST /transcribes request.
// It creates a transcribe of the recording and returns the result.
// @Summary Create a transcribe
// @Description transcribe a recording
// @Produce json
// @Success 200 {object} transcribe.Transcribe
// @Router /v1.0/transcribes [post]
func transcribesPOST(c *gin.Context) {

	var requestParam request.BodyTranscribesPOST

	if err := c.BindJSON(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("transcribesPOST. Received request detail. recording_id: %s, language: %s", requestParam.RecordingID, requestParam.Language)

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create a transcribe
	res, err := serviceHandler.TranscribeCreate(&u, requestParam.RecordingID, requestParam.Language)
	if err != nil {
		logrus.Errorf("Could not create a transcribe. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
