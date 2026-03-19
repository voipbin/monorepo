package billing

import (
	"context"
	"errors"
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
		"request_address": c.ClientIP(),
	})

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.Billing(ctx, c.Request); err != nil {
		log.Errorf("Could not handle the billing webhook. err: %v", err)

		// Return 400 for validation errors (bad signature, malformed body) so Paddle does not retry.
		// Return 500 for transient errors (RPC failure) so Paddle retries later.
		var valErr *servicehandler.ValidationError
		if errors.As(err, &valErr) {
			c.AbortWithStatus(http.StatusBadRequest)
		} else {
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	c.AbortWithStatus(200)
}
