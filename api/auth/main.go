package auth

import (
	"github.com/gin-gonic/gin"
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
// 	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
// 	user, err := serviceHandler.UserCreate(body.Username, body.Password)
// 	if err != nil {
// 		c.AbortWithStatus(400)
// 		return
// 	}

// 	c.JSON(200, user)
// }
