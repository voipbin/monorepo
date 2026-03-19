package billing

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"monorepo/bin-hook-manager/api/models/common"
	"monorepo/bin-hook-manager/pkg/servicehandler"
)

// billingPaddlePOST handles POST /billing/paddle request.
func billingPaddlePOST(c *gin.Context) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":            "billingPaddlePOST",
		"request_address": c.ClientIP,
	})

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.Billing(ctx, c.Request); err != nil {
		log.Errorf("Could not handle the billing webhook. err: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(200)
}
