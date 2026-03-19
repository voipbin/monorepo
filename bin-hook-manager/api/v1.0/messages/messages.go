package messages

import (
	"context"
	"net/http"

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
	log := logrus.WithFields(logrus.Fields{
		"func":            "messagesPOST",
		"request_address": c.ClientIP,
	})

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.Message(ctx, c.Request); err != nil {
		log.Errorf("Could not handle the message correctly. err: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(200)
}
