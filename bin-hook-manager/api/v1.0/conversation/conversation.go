package conversation

import (
	"context"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"monorepo/bin-hook-manager/api/models/common"
	"monorepo/bin-hook-manager/pkg/servicehandler"
)

// conversationPOST handles POST /conversation/... request.
// @Summary deleiver the message to the conversation-manager
// @Description deleiver the message to the message-manager
// @Produce  json
// @Success 200
// @Router /v1.0/conversation [post]
func conversationPOST(c *gin.Context) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "customersIDPOST",
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
	if errHandler := serviceHandler.Conversation(ctx, c.Request.Host+c.Request.URL.Path, data); errHandler != nil {
		log.Errorf("Could not handle the message correctly. err: %v", errHandler)
		c.AbortWithStatus(500)
		return
	}

	c.AbortWithStatus(200)
}
