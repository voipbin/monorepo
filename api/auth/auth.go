package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	// auth.POST("/register", middleware.Authorized, register)
	auth.POST("/login", login)
}

// func register(c *gin.Context) {
// 	logrus.Debug("registering a new user.")

// 	type RequestBody struct {
// 		Username string `json:"username" binding:"required"`
// 		Password string `json:"password" binding:"required"`
// 	}

// 	var body RequestBody
// 	if err := c.BindJSON(&body); err != nil {
// 		c.AbortWithStatus(400)
// 		return
// 	}

// 	// create an user
// 	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
// 	user, err := serviceHandler.UserCreate(body.Username, body.Password)
// 	if err != nil {
// 		c.AbortWithStatus(400)
// 		return
// 	}

// 	c.JSON(200, user)
// }

func login(c *gin.Context) {
	type RequestBody struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	var body RequestBody
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	log := logrus.WithFields(logrus.Fields{
		"username": body.Username,
	})
	log.Debugf("Logging in.")

	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	token, err := serviceHandler.AuthLogin(body.Username, body.Password)
	if err != nil {
		log.Debugf("Login failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.SetCookie("token", token, 60*60*24*7, "/", "", false, true)
	c.JSON(200, map[string]interface{}{
		"username": body.Username,
		"token":    token,
	})

}
