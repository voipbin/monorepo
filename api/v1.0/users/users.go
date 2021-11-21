package users

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// usersPOST creates a new user
func usersPOST(c *gin.Context) {

	var body RequestBodyUsersPOST
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// check permission
	// only admin permssion can create a new user.
	if !u.HasPermission(user.PermissionAdmin) {
		log.Info("The user has no permission")
		c.AbortWithStatus(403)
		return
	}

	// create an user
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	user, err := serviceHandler.UserCreate(body.Username, body.Password, body.Permission)
	if err != nil {
		c.AbortWithStatus(403)
		return
	}

	c.JSON(200, user)
}

// usersGET gets users
func usersGET(c *gin.Context) {

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}

	var requestParam request.ParamUsersGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("usersGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	u := tmp.(user.User)
	log = log.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// check permission
	// only admin permssion users are allowed.
	if !u.HasPermission(user.PermissionAdmin) {
		log.Info("The user has no permission")
		c.AbortWithStatus(403)
		return
	}

	// create an user
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	users, err := serviceHandler.UserGets(requestParam.PageSize, requestParam.PageToken)
	if err != nil {
		log.Errorf("Could not get users info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, users)
}
