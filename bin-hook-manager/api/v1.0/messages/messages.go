package messages

import (
	"context"
	"errors"
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

		// a callback that is refused (no signature, a signature that the message-manager refused, a body that is too
		// large) is a client error and not a server fault. Every other error keeps the 500.
		var valErr *servicehandler.ValidationError
		if errors.As(err, &valErr) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(200)
}
