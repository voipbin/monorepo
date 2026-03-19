package conversation

import (
	"context"

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

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.Conversation(ctx, c.Request); err != nil {
		log.Errorf("Could not handle the message correctly. err: %v", err)
		c.AbortWithStatus(500)
		return
	}

	c.AbortWithStatus(200)
}
