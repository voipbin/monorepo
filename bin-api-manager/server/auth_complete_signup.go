package server

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/gens/openapi_server"
)

func (h *server) PostAuthCompleteSignup(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthCompleteSignup",
		"request_address": c.ClientIP(),
	})
	log.Debug("Processing complete signup.")

	var req openapi_server.PostAuthCompleteSignupJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CustomerCompleteSignup(c.Request.Context(), req.TempToken, req.Code)
	if err != nil {
		log.Debugf("Complete signup failed. err: %v", err)
		if err.Error() == "too many attempts" {
			c.AbortWithStatus(429)
			return
		}
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
