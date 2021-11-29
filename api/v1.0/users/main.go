package users

import "github.com/gin-gonic/gin"

// ApplyRoutes applies router to the gin Engine
func ApplyRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")

	users.POST("", usersPOST)
	users.GET("", usersGET)

	users.GET("/:id", usersIDGET)
	users.PUT("/:id", usersIDPUT)

	users.PUT("/:id/password", usersIDPasswordPUT)
	users.PUT("/:id/permission", usersIDPermissionPUT)
}
