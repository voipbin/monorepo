package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *server) PostAuthLogin(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthLogin",
		"request_address": c.ClientIP,
	})

	var req openapi_server.PostAuthLoginJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log = log.WithFields(logrus.Fields{
		"username": req.Username,
	})
	log.Debugf("Logging in. username: %s", req.Username)

	token, err := h.serviceHandler.AuthLogin(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		log.Debugf("Login failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Created token string. token: %v", token)

	c.SetCookie("token", token, int(servicehandler.TokenExpiration.Seconds()), "/", "", false, true)
	res := openapi_server.AuthLoginResponse{
		Username: req.Username,
		Token:    token,
	}
	log.Debug("User successfully logged in.")

	c.JSON(200, res)
}
