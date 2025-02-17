package service

import (
	// "monorepo/bin-api-manager/api/models/request"
	// "monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type RequestBodyLoginPOST struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// BodyLoginPOST is response body define for POST /login
type ResponseBodyLoginPOST struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

// PostLogin handles POST /PostLogin request.
// It generates and return the JWT token for api use.
//
//	@Summary		Generate the JWT token and return it.
//	@Description	Generate the JWT token and return it.
//	@Produce		json
//	@Param			login_info	body		request.BodyLoginPOST	true	"login info"
//	@Success		200			{object}	response.BodyLoginPOST
//	@Router			/auth/login [post]
func PostLogin(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostLogin",
		"request_address": c.ClientIP,
	})

	var req RequestBodyLoginPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log = log.WithFields(logrus.Fields{
		"username": req.Username,
	})
	log.Debugf("Logging in. username: %s", req.Username)

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	token, err := serviceHandler.AuthLogin(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		log.Debugf("Login failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Created token string. token: %v", token)

	c.SetCookie("token", token, int(servicehandler.TokenExpiration.Seconds()), "/", "", false, true)
	res := ResponseBodyLoginPOST{
		Username: req.Username,
		Token:    token,
	}
	log.Debug("User successfully logged in.")

	c.JSON(200, res)
}
