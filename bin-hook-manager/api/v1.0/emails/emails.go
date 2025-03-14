package emails

import (
	"context"
	"io"
	"monorepo/bin-hook-manager/api/models/common"
	"monorepo/bin-hook-manager/pkg/servicehandler"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// emailsPOST handles POST /emails request.
// @Summary deleiver the message to the email-manager
// @Description deleiver the message to the email-manager
// @Produce  json
// @Success 200
// @Router /v1.0/emails [post]
func emailsPOST(c *gin.Context) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":            "emailsPOST",
		"request_address": c.ClientIP,
	})

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Errorf("Could not read the data. err: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if errHandler := serviceHandler.Email(ctx, c.Request.Host+c.Request.URL.Path, data); errHandler != nil {
		log.Errorf("Could not handle the message correctly. err: %v", errHandler)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(200)
}
