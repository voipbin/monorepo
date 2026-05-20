package conversation

import (
	"context"
	"net/http"

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
	challenge, err := serviceHandler.Conversation(ctx, c.Request)
	if err != nil {
		log.Errorf("Could not handle the message correctly. err: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if challenge != "" {
		c.String(http.StatusOK, challenge)
		return
	}

	c.AbortWithStatus(http.StatusOK)
}
