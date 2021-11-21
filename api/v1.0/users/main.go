package users

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")

	users.POST("", usersPOST)
	users.GET("", usersGET)
}

// RequestBodyUsersPOST is rquest body define for POST /users
type RequestBodyUsersPOST struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Permission uint64 `json:"permission"`
}
