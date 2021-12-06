package users

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// usersPOST handles POST /users request.
// It creates a new user.
// @Summary Create a new user.
// @Description create a new user.
// @Produce  json
// @Param user body request.BodyUsersPOST true "Creating user info."
// @Success 200 {object} user.User
// @Router /v1.0/users [post]
func usersPOST(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "usersPOST",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	var req request.BodyUsersPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not marshal the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing usersPOST.")

	// create an user
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.UserCreate(&u, req.Username, req.Password, req.Name, req.Detail, user.Permission(req.Permission))
	if err != nil {
		log.Errorf("Could not create the use. err: %v", err)
		c.AbortWithStatus(403)
		return
	}

	c.JSON(200, res)
}

// usersGET handles GET /users request.
// It returns list of tags of the given user.
// @Summary List tags
// @Description get tags of the user
// @Produce  json
// @Param page_size query int false "The size of results. Max 100"
// @Param page_token query string false "The token. tm_create"
// @Success 200 {object} response.BodyUsersGET
// @Router /v1.0/users [get]
func usersGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "usersGET",
			"request_address": c.ClientIP,
			"request_uri":     c.Request.URL,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	var req request.ParamUsersGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing usersGET.")

	// get tmps
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	tmps, err := serviceHandler.UserGets(&u, req.PageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get users info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyUsersGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// usersIDGET handles GET /users/<user-id> request.
// It gets the user.
// @Summary Get the user
// @Description Get the user of the given id
// @Produce json
// @Param id path string true "The user's id."
// @Success 200
// @Router /v1.0/users/{id} [get]
func usersIDGET(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "usersIDGET",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	// get id
	tmpID := c.Params.ByName("id")
	id, _ := strconv.Atoi(tmpID)
	log.WithField("request", c.Request.Body).Debug("Executing usersIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.UserGet(&u, uint64(id))
	if err != nil {
		log.Errorf("Could not get users info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// usersIDDELETE handles DELETE /users/<user-id> request.
// It delets the user.
// @Summary Delete the user
// @Description Delete the user of the given id
// @Produce json
// @Param id path string true "The user's id."
// @Success 200
// @Router /v1.0/users/{id} [delete]
func usersIDDELETE(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "usersIDDELETE",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	// get id
	tmpID := c.Params.ByName("id")
	id, _ := strconv.Atoi(tmpID)
	log.WithField("request_body", c.Request.Body).Debug("Executing usersIDDELETE.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.UserDelete(&u, uint64(id)); err != nil {
		log.Errorf("Could not delete the user info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// usersIDPUT handles PUT /users/<user-id> request.
// It updates the user's basic info.
// @Summary Put the user
// @Description Get the user of the given id
// @Produce json
// @Param id path string true "The ID of the user"
// @Param update_info body request.BodyUsersIDPUT true "The update info."
// @Success 200
// @Router /v1.0/users/{id} [put]
func usersIDPUT(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "usersIDPUT",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	var req request.BodyUsersIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.WithField("request", req).Debug("Executing usersIDPUT.")

	// get id
	tmpID := c.Params.ByName("id")
	id, _ := strconv.Atoi(tmpID)
	log.WithField("request_body", c.Request.Body).Debug("Received request detail.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.UserUpdate(&u, uint64(id), req.Name, req.Detail); err != nil {
		log.Errorf("Could not update user's basic info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// usersIDPasswordPUT handles PUT /users/<user-id>/password request.
// It updates the user's password.
// @Summary Put the user
// @Description Get the user of the given id
// @Produce json
// @Param id path string true "The user's id."
// @Param update_info body request.BodyUsersIDPasswordPUT true "Update info."
// @Success 200
// @Router /v1.0/users/{id}/password [put]
func usersIDPasswordPUT(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "usersIDPasswordPUT",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	var req request.BodyUsersIDPasswordPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not marshal the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get id
	tmpID := c.Params.ByName("id")
	id, _ := strconv.Atoi(tmpID)
	log.WithField("request_body", c.Request.Body).Debug("Received request detail.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.UserUpdatePassword(&u, uint64(id), req.Password); err != nil {
		log.Errorf("Could not update user's basic info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}

// usersIDPermissionPUT handles PUT /users/<user-id>/permission request.
// It updates the user's permission.
// @Summary Put the user
// @Description Get the user of the given id
// @Produce json
// @Param id path string true "The user's id."
// @Param update_info body request.BodyUsersIDPermissionPUT true "Update info."
// @Success 200
// @Router /v1.0/users/{id}/permission [put]
func usersIDPermissionPUT(c *gin.Context) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":            "usersIDPermissionPUT",
			"request_address": c.ClientIP,
		},
	)

	tmp, exists := c.Get("user")
	if !exists {
		log.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(user.User)
	log = log.WithFields(
		logrus.Fields{
			"user_id":    u.ID,
			"username":   u.Username,
			"permission": u.Permission,
		},
	)

	var req request.BodyUsersIDPermissionPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not marshal the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get id
	tmpID := c.Params.ByName("id")
	id, _ := strconv.Atoi(tmpID)
	log.WithField("request_body", c.Request.Body).Debug("Received request detail.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.UserUpdatePermission(&u, uint64(id), user.Permission(req.Permission)); err != nil {
		log.Errorf("Could not update user's permission. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
}
