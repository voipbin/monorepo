package tts

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/buckethandler"
)

// ttsGET returns tts wav file
func ttsGET(c *gin.Context) {

	// get target
	filename := c.Params.ByName("filename")

	log := logrus.WithFields(
		logrus.Fields{
			"target": filename,
		},
	)

	target := fmt.Sprintf("%s/%s", "tts", filename)
	log = log.WithFields(
		logrus.Fields{
			"target": target,
		},
	)
	log.Debug("Getting tts file.")

	// get service
	bucketHandler := c.MustGet(common.OBJServiceHandler).(buckethandler.BucketHandler)

	// get file
	data, err := bucketHandler.FileGet(target)
	if err != nil {
		log.Errorf("Could not get file. err: %v", err)
		c.AbortWithStatus(400)
	}

	c.Data(200, "audio/wav", data)

}
