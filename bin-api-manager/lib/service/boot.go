package service

import (
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestBodyBootPOST is request body for POST /boot
type RequestBodyBootPOST struct {
	DirectHash string `json:"direct_hash" binding:"required"`
}

// PostBoot handles POST /boot request.
// It resolves a direct hash and returns a resource-scoped JWT.
func PostBoot(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostBoot",
		"request_address": c.ClientIP(),
	})

	var req RequestBodyBootPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log = log.WithFields(logrus.Fields{
		"direct_hash": req.DirectHash,
	})
	log.Debugf("Processing boot request.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.AuthBoot(c.Request.Context(), req.DirectHash)
	if err != nil {
		log.Infof("Boot failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log.Debug("Boot successful.")
	c.JSON(200, res)
}
