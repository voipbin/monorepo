package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// loginPost handles POST /loginPost request.
// It generates and return the JWT token for api use.
//	@Summary		Generate the JWT token and return it.
//	@Description	Generate the JWT token and return it.
//	@Produce		json
//	@Param			login_info	body		request.BodyLoginPOST	true	"login info"
//	@Success		200			{object}	response.BodyLoginPOST
//	@Router			/auth/login [post]
func loginPost(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "loginPost",
		"request_address": c.ClientIP,
	})

	var req request.BodyLoginPOST
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

	// 1 week
	c.SetCookie("token", token, 60*60*24*7, "/", "", false, true)
	res := response.BodyLoginPOST{
		Username: req.Username,
		Token:    token,
	}
	log.Debug("User successfully logged in.")

	c.JSON(200, res)
}
