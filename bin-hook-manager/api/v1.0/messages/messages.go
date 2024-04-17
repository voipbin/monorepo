package messages

import (
	"context"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"monorepo/bin-hook-manager/api/models/common"
	"monorepo/bin-hook-manager/pkg/servicehandler"
)

// messagesPOST handles POST /messages request.
// @Summary deleiver the message to the message-manager
// @Description deleiver the message to the message-manager
// @Produce  json
// @Success 200
// @Router /v1.0/messages [post]
func messagesPOST(c *gin.Context) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "messagesPOST",
			"request_address": c.ClientIP,
		},
	)

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Errorf("Could not read the data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if errHandler := serviceHandler.Message(ctx, c.Request.Host+c.Request.URL.Path, data); errHandler != nil {
		log.Errorf("Could not handle the message correctly. err: %v", errHandler)
		c.AbortWithStatus(500)
		return
	}

	c.AbortWithStatus(200)
}
